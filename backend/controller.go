package backend

import (
	"context"
	"encoding/json"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"
)

const CONTENT_TYPE_KEY = "Content-Type"
const CONTENT_TYPE_VALUE = "application/json"

var version string

func init() {
}

func StartService(versionMajor, versionMin string) {
	Logger.Warnf("server start, version: %s.%s", versionMajor, versionMin)
	version = versionMajor + "." + versionMin
	Logger.Debugf("port=%s", *Addr)

	r := newRouter()

	srv := &http.Server{
		Addr: *Addr,
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      r, // Pass our instance of gorilla/mux in.
	}

	// Run our server in a goroutine so that it doesn't block.
	go func() {
		var err error
				
		if *EnableTLS {
			Logger.Info("server started with TLS")
			err = srv.ListenAndServeTLS(*SSLcertFile, *SSLkeyFile);
		} else {
			Logger.Info("server started")
			err = srv.ListenAndServe();
		}
		
		if err != nil {
			Logger.Fatal(err)
		}
	}()

	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal.
	<-c

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), *Timeout)
	defer cancel()
	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	srv.Shutdown(ctx)
	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	closeDB()
	Logger.Info("server shutting down")
	os.Exit(0)
}

// The new router function creates the router and
// returns it to us. We can now use this function
// to instantiate and test the router outside of the main function
func newRouter() *mux.Router {
	r := mux.NewRouter()
	r.Use(handlerMiddleware)

	r.HandleFunc("/channels", getPlaylistsHandler).Methods("GET")

	if *ListenAdmin {
		routeAdminPlaylist := r.PathPrefix("/channels/admin").Subrouter()
		routeAdminPlaylist.Methods("GET").HandlerFunc(getPlaylistsHandlerAdmin)
		routeAdminPlaylist.Methods("OPTIONS", "POST").HandlerFunc(appendPlaylistHandler)
		routeAdminPlaylist.Path("/{id}").Methods("OPTIONS", "PUT").HandlerFunc(updatePlaylistHandler)
		routeAdminPlaylist.Path("/{id}").Methods("OPTIONS", "DELETE").HandlerFunc(deletePlaylistHandler)
	}

	routeVideo := r.PathPrefix("/view").Subrouter()
	routeVideo.Path("/counts").Methods("GET").HandlerFunc(getGlobalCountsHandler)
	routeVideo.Path("/videos").Methods("GET").HandlerFunc(getVidesHandler)
	routeVideo.Path("/videos/{id}").Methods("GET").HandlerFunc(getVideoByIdChannelHandler)
	routeVideo.Path("/video/{id}").Methods("GET").HandlerFunc(getVideoByIdHandler)
	routeVideo.Path("/metrics/{id}").Methods("GET").HandlerFunc(getMetricsByVideoIdHandler)

	printRouter(r)

	return r
}

// print all of the routes that are registered on a router
func printRouter(r *mux.Router) {
	err := r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		pathTemplate, err := route.GetPathTemplate()
		if err == nil {
			Logger.Debugf("ROUTE=%s", pathTemplate)
		}
		pathRegexp, err := route.GetPathRegexp()
		if err == nil {
			Logger.Debugf("Path regexp=%s", pathRegexp)
		}
		queriesTemplates, err := route.GetQueriesTemplates()
		if err == nil {
			Logger.Debugf("Queries templates=%s", strings.Join(queriesTemplates, ","))
		}
		queriesRegexps, err := route.GetQueriesRegexp()
		if err == nil {
			Logger.Debugf("Queries regexps=%s", strings.Join(queriesRegexps, ","))
		}
		methods, err := route.GetMethods()
		if err == nil {
			Logger.Debugf("Methods=%s", strings.Join(methods, ","))
		}
		return nil
	})

	if err != nil {
		Logger.Debug(err)
	}
}

// Перехоплення та обробка всіх запитів
func handlerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method := r.Method
		origin := r.Header.Get("Origin")
		Logger.Infof("method: %v, uri: %v, addr: %v, origin: [%v], host: %v", method, r.RequestURI, r.RemoteAddr, origin, r.Host)

		// Перевірка на валідний Origin  
		if origin == *Origin {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

			if method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
			} else {
				// Call the next handler, which can be another middleware in the chain, or the final handler.
				next.ServeHTTP(w, r)
			}
		} else {
			Logger.Errorf("invalid origin: [%v] != [%v] ",origin, *Origin)
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}

	})
}

// Парсинг тіла запиту
func parseBody(r *http.Request, i interface{}) error {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	Logger.Debugf("body=%v", string(b))

	err = json.Unmarshal(b, i)
	if err != nil {
		return err
	}

	return nil
}

// Оброблювач запиту на додавання плей-листа
func appendPlaylistHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	req := q.Get("req")
	Logger.Debugf("req=%v(%v)", req, formatStringDate(req))

	var channel Channel
	err := parseBody(r, &channel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	Logger.Debugf("channel=%v", channel)

	err = addChannel(&channel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	Logger.Warnf("appended channel=%v", channel)
}

// Оброблювач запиту на оновлення плей-листа
func updatePlaylistHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	q := r.URL.Query()
	req := q.Get("req")
	Logger.Debugf("req=%v(%v)", req, formatStringDate(req))

	var channel Channel
	err := parseBody(r, &channel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	Logger.Debugf("channel=%v", channel)

	err = updateChannel(id, &channel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	Logger.Infof("updated channel=%v", channel)
}

// Оброблювач запиту на видалення плей-листа
func deletePlaylistHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	q := r.URL.Query()
	req := q.Get("req")

	Logger.Debugf("req=%v(%v)", req, formatStringDate(req))

	err := deleteChannel(id)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	Logger.Infof("deleted channel with id=%v", id)
}

func formatStringDate(sdt string) string {
	t, err := strconv.ParseInt(sdt, 10, 64)
	if err != nil {
		return sdt
	}

	return time.Unix(0, t*int64(time.Millisecond)).String()
}

// Оброблювач запиту на отримання всіх активних плей-листів
func getPlaylistsHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	req := q.Get("req")
	enable := q.Get("enable")
	Logger.Debugf("req=%v(%v), active=%v", req, formatStringDate(req), enable)

	channelJson, err := getPlaylists(true)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(CONTENT_TYPE_KEY, CONTENT_TYPE_VALUE)
	//		w.Header().Set(CONTENT_LENGTH_KEY, strconv.Itoa(len(channelJson)))

	w.WriteHeader(http.StatusOK)
	w.Write(channelJson)
}

// Оброблювач запиту на отримання всіх плей-листів для для адміністрування
func getPlaylistsHandlerAdmin(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	req := q.Get("req")
	Logger.Debugf("req=%v(%v)", req, formatStringDate(req))

	channelJson, err := getPlaylists(false)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(CONTENT_TYPE_KEY, CONTENT_TYPE_VALUE)
	//		w.Header().Set(CONTENT_LENGTH_KEY, strconv.Itoa(len(channelJson)))

	w.WriteHeader(http.StatusOK)
	w.Write(channelJson)
}

// Оброблювач запиту на отриматння метрик по відео id за заданий період
func getMetricsByVideoIdHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	id := vars["id"]
	if id == "" {
		http.Error(w, "video id is null", http.StatusInternalServerError)
		return
	}

	q := r.URL.Query()
	req := q.Get("req")
	from := q.Get("from")
	to := q.Get("to")
	Logger.Debugf("req=%v(%v), id=%v, from=%v, to=%v", req, formatStringDate(req), id, from, to)

	metricsVideoJson, err := getMetricsById(id, from, to)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(CONTENT_TYPE_KEY, CONTENT_TYPE_VALUE)
	//	w.Header().Set(CONTENT_LENGTH_KEY, strconv.Itoa(len(metricsVideoJson)))

	w.WriteHeader(http.StatusOK)
	w.Write(metricsVideoJson)
}

// Оброблювач запиту даних по відео id
func getVideoByIdHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	id := vars["id"]
	if id == "" {
		http.Error(w, "video id is null", http.StatusInternalServerError)
		return
	}

	q := r.URL.Query()
	req := q.Get("req")
	Logger.Debugf("id: %v, req: %v(%v)", id, req, formatStringDate(req))

	videoJson, err := getVideoById(id)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(CONTENT_TYPE_KEY, CONTENT_TYPE_VALUE)
	//	w.Header().Set(CONTENT_LENGTH_KEY, strconv.Itoa(len(vdeoJson)))

	w.WriteHeader(http.StatusOK)
	w.Write(videoJson)
}


// Оброблювач запиту на отримання списку всіх відео 
func getVidesHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	req := q.Get("req")
	skip := q.Get("skip")

	offset, err := strconv.Atoi(skip)
	if err != nil {
		offset = 0
	}

	Logger.Debugf("req=%v(%v), offset=%v", req, formatStringDate(req), offset)

	videosJson, err := getVideos(offset)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(CONTENT_TYPE_KEY, CONTENT_TYPE_VALUE)
	//	w.Header().Set(CONTENT_LENGTH_KEY, strconv.Itoa(len(videosJson)))

	w.WriteHeader(http.StatusOK)
	w.Write(videosJson)
}


// Оброблювач запиту на отримання списку відео id плейлиста
func getVideoByIdChannelHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	q := r.URL.Query()
	req := q.Get("req")
	skip := q.Get("skip")

	offset, err := strconv.Atoi(skip)
	if err != nil {
		offset = 0
	}

	Logger.Debugf("req=%v(%v), id=%v, offset=%v", req, formatStringDate(req), id, offset)

	videosJson, err := getVideosByChannelId(id, offset)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(CONTENT_TYPE_KEY, CONTENT_TYPE_VALUE)
	//	w.Header().Set(CONTENT_LENGTH_KEY, strconv.Itoa(len(videosJson)))

	w.WriteHeader(http.StatusOK)
	w.Write(videosJson)
}


// Оброблювач запиту на отримання глобальних метрик (кількість відео, кількість плейлистів)
func getGlobalCountsHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	req := q.Get("req")
	Logger.Debugf("req=%v", req)

	globalCountsJson, err := getGlobalCounts(version)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(CONTENT_TYPE_KEY, CONTENT_TYPE_VALUE)
	//	w.Header().Set(CONTENT_LENGTH_KEY, strconv.Itoa(len(videosJson)))

	w.WriteHeader(http.StatusOK)
	w.Write(globalCountsJson)
}

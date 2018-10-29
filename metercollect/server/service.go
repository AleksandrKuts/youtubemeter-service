package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/youtube/v3"

	"github.com/AleksandrKuts/go/youtubemeter/metercollect/config"
	"github.com/AleksandrKuts/go/youtubemeter/metercollect/server/database"
)

const LAYOUT_ISO_8601 = "2006-01-02T15:04:05Z"
const PLAY_LIST_MAX_RESULT = 20
const PLAY_LIST_PART = "snippet,contentDetails"
const CHANNEL_PART = "snippet,contentDetails,statistics"
const VIDEOS_PART = "snippet,contentDetails,statistics"

const missingClientSecretsMessage = `
Please configure OAuth 2.0
`

var service *youtube.Service

func init() {
	ctx := context.Background()

	b, err := ioutil.ReadFile(*config.FileSecret)
	if err != nil {
		log.Fatalf("Unable to read client secret file. %v", err)
	}

	config, err := google.ConfigFromJSON(b, youtube.YoutubeReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}

	client := getClient(ctx, config)
	service, err = youtube.New(client)
	if err != nil {
		log.Fatalf("Error creating YouTube client: %v", err)
	}

}

func StartService() {
	log.Warn("server start")

	checkPlayLists()
	checkVideos()
	getMeters()

	timerPlayList := time.Tick(*config.PeriodPlayList)
	timerVideo := time.Tick(*config.PeriodVideo)
	timerMeter := time.Tick(*config.PeriodMeter)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	for {
		select {
		case <-timerPlayList:
			go checkPlayLists()
		case <-timerVideo:
			go checkVideos()
		case <-timerMeter:
			go getMeters()
		case <-quit:
			log.Warn("Service shutting down")
			return
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}
}

// getClient uses a Context and Config to retrieve a Token
// then generate a Client. It returns the generated Client.
func getClient(ctx context.Context, config *oauth2.Config) *http.Client {
	cacheFile, err := tokenCacheFile()
	if err != nil {
		log.Fatalf("Unable to get path to cached credential file. %v", err)
	}
	tok, err := tokenFromFile(cacheFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(cacheFile, tok)
	}

	log.Debugf("token=%v", tok)
	return config.Client(ctx, tok)
}

// getTokenFromWeb uses Config to request a Token.
// It returns the retrieved Token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	log.Warnf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		log.Fatalf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(oauth2.NoContext, code)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
	}
	return tok
}

// tokenCacheFile generates credential file path/filename.
// It returns the generated credential path/filename.
func tokenCacheFile() (string, error) {
//	usr, err := user.Current()
//	if err != nil {
//		return "", err
//	}
//	tokenCacheDir := filepath.Join(usr.HomeDir, ".credentials")
//	os.MkdirAll(tokenCacheDir, 0700)
//	return filepath.Join(tokenCacheDir,
//		url.QueryEscape("youtube-metrics.json")), err

	if _, err := os.Stat( *config.CredentialFile ); err != nil {
		return "", err
	} else {
		return *config.CredentialFile, nil;	
	}
}

// tokenFromFile retrieves a Token from a given file path.
// It returns the retrieved Token and any read error encountered.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	t := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(t)
	defer f.Close()

	return t, err
}

// saveToken uses a file path to create a file and store the
// token in it.
func saveToken(file string, token *oauth2.Token) {
	log.Warnf("Saving credential file to: %s\n", file)
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

// Перевіряємо список плейлистів, чи додав адміністратор нові, чи видалив, чи деактивував, та корегуємо
func checkPlayLists() {
	log.Debug("check playlist")

	// Отримуємо перечень діючих PlayList-ів з БД на даний час
	ids, err := database.GetPlaylistIDs()

	if err != nil {
		log.Errorf("Error get id's playlists: ", err)
		return
	}

	log.Debugf("ids=%v", ids)

	if ids != nil && len(ids) > 0 {

		playlists.mux.Lock()
		defer playlists.mux.Unlock()

		// Перевіряємо список на видалення чи деактивування
		for id, pl := range playlists.playlists {

			_, ok := ids[id]
			if ok == false { // підлягає припиненню обробки
				if pl.deleted { // вже помічений на припинення обробки
					if time.Since(pl.timeDeleted) > *config.PeriodDeleted { // перевіряємо, чи не час припиняти обробку
						playlists.delete(id) // видалення
						log.Infof("stop processing playlist with id: %v", id)
					}
				} else { // ще не помічений на припинення обробки
					playlists.setDeletedPlayList(id) // помічаємо: підлягае припиненню обробки
					log.Debugf("set stop processing playlist with id: %v", id)
				}
			} else { // не підлягае видаленню
				if pl.deleted { // але раніше підлягав
					playlists.canselDeletedPlayList(id) // відміна видалення
					log.Debugf("cansel stop processing playlist with id: %v", id)
				}
			}

		}

		// Перевіряємо список на додавання нових плейлистів
		for id, _ := range ids {
			_, ok := playlists.playlists[id]
			if ok == false {
				playlists.append(id) // додаемо новий PlayList
				log.Infof("Append playlist with id: %v", id)
			}
		}

	}
}

// Отримати тимчасовий список плейлистів для роботи з сервісами Youtube. Цей тимчасовий список потрібен щоб не
// блокувати надовго роботу з основним списком, в якій можуть додати, або видалити плейлист
// Плейлисти помічені на видалення ігноруються
func getRequestPlayList() map[string]*YoutubePlayList {
	requestPlayList := make(map[string]*YoutubePlayList)

	// блокування потрібно щоб гарантовано не почати обробляти PlayList якій видалений, чи деактивований
	playlists.mux.Lock()
	defer playlists.mux.Unlock()
	for id, playList := range playlists.playlists {
		if !playList.deleted { // додаються тільки робочі плейлисти
			requestPlayList[id] = playList
		}
	}

	return requestPlayList
}

// Перевіряємо список відео в плейлистах, чи були додані нові, чи вичерпався термін збору статистики на старих
func checkVideos() {
	log.Debug("check videos start")

	requestPlayList := getRequestPlayList() // отримуємо список плейлистів для запросів
	log.Debugf("request play list: %v", requestPlayList)

	for id, playList := range requestPlayList {
		go checkVideosByPlaylistId(id, playList)
	}

	log.Debug("check videos end")
}

// Перевіряємо список відео конкретного плейлиста, чи були додані нові, чи вичерпався термін збору статистики на старих
// для отримання списку відео викоритовується сервіс https://developers.google.com/youtube/v3/docs/playlistItems
func checkVideosByPlaylistId(playListId string, playList *YoutubePlayList) {
	// перевіряємо плейлист на застаріле відео яке вже не потрібно обробляти
	checkElapsedVideos(playList)

	if playListId == "" {
		log.Error("Error, PlayList id is empty")
		return
	}

	call := service.PlaylistItems.List(PLAY_LIST_PART)
	call = call.MaxResults(PLAY_LIST_MAX_RESULT)
	call = call.PlaylistId(playListId)
	response, err := call.Do()
	if err != nil {
		log.Errorf("Error get play list by id=%v, error=%v", playListId, err)
		return
	}

	// перевіряємо плейлист на появу нових відео
	for _, item := range response.Items {
		videoId := item.ContentDetails.VideoId

		log.Debugf("video: id=%v", videoId)

		_, ok := playList.videos[videoId]
		if ok == false { // такого відео ще нема, пробуємо додати
			addVideo(playListId, playList, videoId, item)
		}
	}
}

// Перевіряє ПлейЛист чи не настав час (задається через config.PeriodСollection) припинити обробку якихось відео
// Спочатку відео помічаєтеся для видалення, а через заданий час (config.PeriodDeleted) видаляється остаточно
// Рознесення в часі помітки відео на видалення і само видалення гарантує коректну роботу потоків програми
func checkElapsedVideos(playList *YoutubePlayList) {
	playList.mux.Lock()
	defer playList.mux.Unlock()
	for id, video := range playList.videos {

		if video.deleted { // якщо відео призначене для видалення
			if time.Since(video.timeDeleted) > *config.PeriodDeleted { // перевіряємо, чи не час видаляти
				playList.delete(id) // видалення
				log.Infof("stop processing video with id: %v", id)
			}
		} else { // відео ще не призначене для видалення
			// Перевірка чи не потрібно припинити обробку відео за часом
			if time.Since(video.PublishedAt) > *config.PeriodСollection {
				playList.setDeletedVideo(id)
				log.Infof("set stop processing video with id: %v", id)
			}

		}
	}
}

// додаємо відео для збору статистики
func addVideo(playListId string, playList *YoutubePlayList, videoId string, item *youtube.PlaylistItem) {
	if videoId == "" {
		log.Error("Error, Video id is empty")
		return
	}

	title := item.Snippet.Title
	publishedAt := item.Snippet.PublishedAt
	description := item.Snippet.Description
	channelId := item.Snippet.ChannelId
	channelTitle := item.Snippet.ChannelTitle

	// етап перевірки чи не застаріле відео
	timePublishedAt, err := time.Parse(LAYOUT_ISO_8601, publishedAt)
	if err != nil {
		log.Errorf("Error parse PublishedAt %v", publishedAt)
		return
	}
	timeElapsed := time.Since(timePublishedAt)
	if timeElapsed > *config.PeriodСollection {
		log.Debugf("skip video with id: %v, time elapsed: %v", videoId, timeElapsed)
		return
	}

	// відео пройшло перевірку, додаємо його для збору статистики
	err = database.AddVideo(videoId, playListId, timePublishedAt, title, description, channelId, channelTitle )
	if err != nil {
		log.Error(err)
		return
	}

	playList.mux.Lock()
	defer playList.mux.Unlock()
	playList.append(videoId, &YoutubeVideo{PublishedAt: timePublishedAt, Title: title, deleted: false})
	log.Infof("add new video: id=%v, at=%v, title=%v ", videoId, timePublishedAt, title)
}

func getMeters() {
	log.Debug("get meters")

	requestPlayList := getRequestPlayList() // отримуємо список плейлистів для запросів
	log.Debugf("request play list: %v", requestPlayList)

	for id, playList := range requestPlayList {
		go getMetersVideos(id, playList)
	}

}

// Отримати тимчасовий список відео для роботи з сервісами Youtube. Цей тимчасовий список потрібен щоб не
// блокувати надовго роботу з основним списком, в якій можуть додати, або видалити відео
// Відео помічені на видалення ігноруються
func getRequestVideosFromPlayList(playList *YoutubePlayList) (string, map[string]*YoutubeVideo) {
	requestVideos := make(map[string]*YoutubeVideo)

	// блокування потрібно щоб гарантовано не почати обробляти PlayList якій видалений, чи деактивований
	playList.mux.Lock()
	defer playList.mux.Unlock()

	var bIds bytes.Buffer
	var isFirst = true

	for id, video := range playList.videos {
		if !video.deleted { // додаються тільки робочі плейлисти
			if isFirst {
				isFirst = false
			} else {
				bIds.WriteString(",")
			}
			bIds.WriteString(id)

			requestVideos[id] = video
		}
	}

	return bIds.String(), requestVideos
}

func getMetersVideos(id string, playList *YoutubePlayList) {
	log.Debugf("getMetersVideo")

	ids, requestVideos := getRequestVideosFromPlayList(playList)

	call := service.Videos.List(VIDEOS_PART)
	call = call.Id(ids)

	response, err := call.Do()
	if err != nil {
		log.Errorf("Error get video list by ids=%v, error=%v", ids, err)
		return
	}

	var metrics = []*database.Metrics{}

	for _, item := range response.Items {
		videoId := item.Id
		videoCommentCount := item.Statistics.CommentCount
		videoLikeCount := item.Statistics.LikeCount
		videoDislikeCount := item.Statistics.DislikeCount
		videoViewCount := item.Statistics.ViewCount

		log.Debugf("id=%v, comment=%10v, like=%10v, dislike=%10v, view=%10v",
			videoId,
			videoCommentCount,
			videoLikeCount,
			videoDislikeCount,
			videoViewCount)

		rVideo, ok := requestVideos[videoId]
		if ok == true {
			// Перевірка чи не потрібно припинити обробку відео за часом
			if time.Since(rVideo.timeCount) > *config.PeriodCount ||
				rVideo.CommentCount != videoCommentCount ||
				rVideo.LikeCount != videoLikeCount ||
				rVideo.DislikeCount != videoDislikeCount ||
				rVideo.ViewCount != videoViewCount {

				rVideo.setMetrics(videoCommentCount, videoLikeCount, videoDislikeCount, videoViewCount)
				metrics = append(metrics, &database.Metrics{videoId, videoCommentCount, videoLikeCount, videoDislikeCount, videoViewCount, time.Now()})
				log.Debugf("save metrics, video id=%v", videoId)
			}
		} else {
			log.Errorf("Cannot get request video with id %v=", videoId)
		}
	}

	if len(metrics) > 0 {
		database.AddMetric(metrics)
		log.Infof("save metrics video for playlist id=%v", id)
	}
}

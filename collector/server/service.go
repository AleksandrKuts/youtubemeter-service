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

	"github.com/AleksandrKuts/youtubemeter-service/collector/config"
	"github.com/AleksandrKuts/youtubemeter-service/collector/server/database"
)

const LAYOUT_ISO_8601 = "2006-01-02T15:04:05Z"
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

func StartService(versionMajor, versionMin string) {
	log.Warnf("server start, version: %s.%s\n", versionMajor, versionMin)

	checkPlayLists()
	checkVideos()
	
	time.Sleep(10 * time.Second);
	
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

	log.Debugf("ids from DB: %v", ids)

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
						log.Infof("pl: %v, stop processing playlist", id)
					}
				} else { // ще не помічений на припинення обробки
					playlists.setDeletedPlayList(id) // помічаємо: підлягае припиненню обробки
					log.Debugf("pl: %v, set stop processing playlist", id)
				}
			} else { // не підлягае видаленню
				if pl.deleted { // але раніше підлягав
					playlists.canselDeletedPlayList(id) // відміна видалення
					log.Debugf("pl: %v, cansel stop processing playlist", id)
				}
			}

		}

		// Перевіряємо список на додавання нових плейлистів
		for id, _ := range ids {
			_, ok := playlists.playlists[id]
			if ok == false {
				playlists.append(id) // додаемо новий PlayList
				log.Infof("pl: %v, Append playlist", id)
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

	for _, playList := range requestPlayList {
		go checkVideosByPlaylistId(playList)
	}

	log.Debug("check videos end")
}

// Перевіряємо список відео конкретного плейлиста, чи були додані нові, чи вичерпався термін збору статистики на старих
// для отримання списку відео викоритовується сервіс https://developers.google.com/youtube/v3/docs/playlistItems
func checkVideosByPlaylistId(playList *YoutubePlayList) {
	log.Debugf("pl: %v, check video start, count videos: %v", playList.id, len(playList.videos))

	// перевіряємо плейлист на застаріле відео яке вже не потрібно обробляти
	if len(playList.videos) > 0 {
		checkElapsedVideos(playList)
	}

	playListId := playList.id;

	call := service.PlaylistItems.List(PLAY_LIST_PART)
	call = call.MaxResults(*config.MaxRequestVideos)
	call = call.PlaylistId(playListId)
	response, err := call.Do()
	if err != nil {
		log.Errorf("pl: %v, Error get play list: %v", playList.id, err)
		return
	}

	// перевіряємо плейлист на появу нових відео
	for _, item := range response.Items {
		videoId := item.ContentDetails.VideoId
		log.Debugf("pl: %v, video: %v, is new?",playList.id, videoId)

		_, ok := playList.videos[videoId]
		if ok == false { // такого відео ще нема, пробуємо додати
			addVideo(playList, videoId, item)
		}
	}	
	log.Debugf("pl: %v, check video end,  count videos: %v", playList.id, len(playList.videos))
}

// Перевіряє ПлейЛист чи не настав час (задається через config.PeriodСollection) припинити обробку якихось відео
// Спочатку відео помічаєтеся для видалення, а через заданий час (config.PeriodDeleted) видаляється остаточно
// Рознесення в часі помітки відео на видалення і само видалення гарантує коректну роботу потоків програми
func checkElapsedVideos(playList *YoutubePlayList) {
	playList.mux.Lock()
	defer playList.mux.Unlock()
	
	countDeleted := 0;
	for id, video := range playList.videos {

		if video.deleted { // якщо відео призначене для видалення
			countDeleted++;
			if time.Since(video.timeDeleted) > *config.PeriodDeleted { // перевіряємо, чи не час видаляти
				playList.delete(id) // видалення
				log.Infof("pl: %v, video: %v, stop processing", playList.id, id)
			}
		} else { // відео ще не призначене для видалення
			// Перевірка чи не потрібно припинити обробку відео за часом
			if time.Since(video.PublishedAt) > *config.PeriodСollection {
				playList.setDeletedVideo(id)
				log.Infof("pl: %v, video: %v, set stop processing", playList.id, id)
			}

		}
	}
	log.Infof("pl: %v, check video elaps, count videos: %v, deleted videos: %v", playList.id, len(playList.videos), 
		countDeleted)
}

// додаємо відео для збору статистики
func addVideo(playList *YoutubePlayList, videoId string, item *youtube.PlaylistItem) {
	playListId := playList.id;
	if videoId == "" {
		log.Errorf("pl: %v, error: video id is empty", playListId)
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
		log.Errorf("pl: %v, error parse PublishedAt %v", playListId, publishedAt)
		return
	}
	timeElapsed := time.Since(timePublishedAt)
	if timeElapsed > *config.PeriodСollection {
		log.Debugf("pl: %v, video: %v, skip proccessing, time elapsed: %v", playListId, videoId, timeElapsed)
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
	log.Infof("pl: %v, video: %v, add new at: %v, title: %v", playListId, videoId, timePublishedAt, title)
}

func getMeters() {
	log.Debug("get meters")

	requestPlayList := getRequestPlayList() // отримуємо список плейлистів для запросів
	log.Debugf("get meters, count request playlists: %v", len(requestPlayList))

	for _, playList := range requestPlayList {
		go getMetersVideos(playList)
	}
}

func getMetersVideos(playList *YoutubePlayList) {
	if len(playList.videos) > 0 {
		mRrequestVideos := getRequestVideosFromPlayList(playList)
		for i := 0; i < len(mRrequestVideos); i++ {
			getMetersVideosInd(playList.id, mRrequestVideos[i])
		} 	
	} else {
		log.Debugf("pl: %v, skip since the number of videos 0", playList.id)
		return
	}
}

// Отримати тимчасовий список відео для роботи з сервісами Youtube. Цей тимчасовий список потрібен щоб не
// блокувати надовго роботу з основним списком, в якій можуть додати, або видалити відео
// Відео помічені на видалення ігноруються
// Список відео в запросі ділимо на частини згідно з дозволеною кількістью youtube api: зараз 50
// Повертаємо массив з частинами запросу кожна по 50 відео 
func getRequestVideosFromPlayList(playList *YoutubePlayList) ([]map[string]*YoutubeVideo) {
	log.Debugf("pl: %v, get request playlist, count video: all: %v", playList.id, len(playList.videos))
	mRequestVideos := []map[string]*YoutubeVideo{}
	
	var requestVideos map[string]*YoutubeVideo // перша частина запросу: перші 50 відео
	requestVideos = make(map[string]*YoutubeVideo)
	
	mRequestVideos = append(mRequestVideos, requestVideos)

	// блокування потрібно щоб гарантовано не почати обробляти PlayList якій видалений, чи деактивований
	playList.mux.Lock()
	defer playList.mux.Unlock()

	count := 0
	countall := 0
	for id, video := range playList.videos {
		if !video.deleted { // додаються тільки робочі плейлисти
			// Таке відео повинно бути видалене
			timeElapsed := time.Since(video.PublishedAt)
			if timeElapsed > *config.PeriodСollection {
				log.Errorf("pl: %v, video: %v, deleted but works, timeElapsed: %v, published: %v", playList.id, id, 
					timeElapsed, video.PublishedAt) 
				continue;
			}
		
			// обробляємо тільки дозволену кількість відео. Запрос ділимо на частини
			// Робимо нову частину запросу: ще 50 відео
			if count >= *config.MaxRequestCountVideoID {
				requestVideos = make(map[string]*YoutubeVideo)
				mRequestVideos = append(mRequestVideos, requestVideos)
				count = 0;
			}		
			requestVideos[id] = video
					
			count++;
			countall++;
		}
	}	
	log.Debugf("pl: %v, get request playlist, count video: all: %v, request: %v", playList.id, len(playList.videos), 
		countall)
	
	return mRequestVideos
}

func getMetersVideosInd(idpl string, requestVideos map[string]*YoutubeVideo) {
	log.Debugf("pl: %v, getMetersVideo, count request videos: %v", idpl, len(requestVideos))
	
	// Формуємо стрічку з id подилену комами
	var bIds bytes.Buffer
	var isFirst = true
	for id, _ := range requestVideos {
			if isFirst {
				isFirst = false
			} else {
				bIds.WriteString(",")
			}
			bIds.WriteString(id)
	}
	ids := bIds.String();

	call := service.Videos.List(VIDEOS_PART)
	call = call.Id(ids)

	response, err := call.Do()
	if err != nil {
		log.Errorf("pl: %v, error get video list by ids=%v, error=%v", idpl, ids, err)
		return
	}

	var metrics = []*database.Metrics{}

	for _, item := range response.Items {
		videoId := item.Id
		videoCommentCount := item.Statistics.CommentCount
		videoLikeCount := item.Statistics.LikeCount
		videoDislikeCount := item.Statistics.DislikeCount
		videoViewCount := item.Statistics.ViewCount

		log.Debugf("pl: %v, video: %v, comment: %5v, like: %6v, dislike: %6v, view: %8v",
			idpl,
			videoId,
			videoCommentCount,
			videoLikeCount,
			videoDislikeCount,
			videoViewCount)

		rVideo, ok := requestVideos[videoId]
		if ok == true {
			
			// Заносимо метрики до БД в двох випадках:
			//   1. якщо пройшов заданий період ( PeriodCount ) 
			//   2. якщо змінилась будь яка метрика (лайки, дізлайки тощо)
			if time.Since(rVideo.timeCount) > *config.PeriodCount ||
				rVideo.CommentCount != videoCommentCount ||
				rVideo.LikeCount != videoLikeCount ||
				rVideo.DislikeCount != videoDislikeCount ||
				rVideo.ViewCount != videoViewCount {

				rVideo.setMetrics(videoCommentCount, videoLikeCount, videoDislikeCount, videoViewCount)
				metrics = append(metrics, &database.Metrics{videoId, videoCommentCount, videoLikeCount, 
						videoDislikeCount, videoViewCount, time.Now()})
				log.Debugf("pl: %v, video: %v, save metrics", idpl, videoId)
			}
		} else {
			log.Errorf("pl: %v, Cannot get request video with id %v=", idpl, videoId)
		}
	}

	if len(metrics) > 0 {
		database.AddMetric(metrics)
	}

	log.Infof("pl: %v, video's metrics - save: %v, skip %v", idpl, len(metrics), 
				len(requestVideos) - len(metrics))
}

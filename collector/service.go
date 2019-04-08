package collector

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/youtube/v3"

	"github.com/peterhellberg/duration"
)

// Details on: https://developers.google.com/youtube/v3/docs/
const SEARCH_LIST_PART = "snippet,id"
const CHANNELS_LIST_PART = "contentDetails,id"
const PLAYLISTITEMS__LIST_PART = "snippet"
const VIDEOS_LIST_PART = "statistics"
const VIDEOS_LIST_PART_DETAILS = "contentDetails"
const LIVE_BROADCAST_CONTENT = "live"

const LAYOUT_ISO_8601 = "2006-01-02T15:04:05Z"

const missingClientSecretsMessage = `
Please configure OAuth 2.0
`

// Список каналів для збору статистики. Список корегується згідно з розкладом (PeriodChannel)
var channels YoutubeChannels

var service *youtube.Service

// Поточна кількість запитів до сервісу youtube.playlistItems. Детальніше дивись опис countRequestPlaylistItems
// в collector.ini
var countRequestPlaylistItems int

func init() {
	ctx := context.Background()

	b, err := ioutil.ReadFile(*FileSecret)
	if err != nil {
		Logger.Fatalf("Unable to read client secret file. %v", err)
	}

	config, err := google.ConfigFromJSON(b, youtube.YoutubeReadonlyScope)
	if err != nil {
		Logger.Fatalf("Unable to parse client secret file to config: %v", err)
	}

	client := getClient(ctx, config)
	service, err = youtube.New(client)
	if err != nil {
		Logger.Fatalf("Error creating YouTube client: %v", err)
	}

}

func StartService(versionMajor, versionMin string) {
	Logger.Warnf("server start, version: %s.%s", versionMajor, versionMin)

	// Налагоджеэмо вихід з програми
	quit := make(chan os.Signal, 1)
	signal.Notify(quit,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	initChannels()

	// Перший прохід заповнюємо дані по плейлистам
	Logger.Info("First check videos")
	checkVideos()
	time.Sleep(10 * time.Second)

	// Другий прохід заповнюємо данні по відео
	// використовуємо youtube.seach
//	countRequestPlaylistItems = *CountRequestPlaylistItems
	Logger.Info("Second check videos")
	checkVideos()
	time.Sleep(10 * time.Second)

	Logger.Info("First check meters")
	getMeters()
	time.Sleep(10 * time.Second)

	// Таймер для періодичної перевірки списку каналів
	timerChannel := time.Tick(*PeriodChannel)
	// Таймер для періодичної перевірки списку відео в каналі
	timerVideo := time.Tick(*PeriodVideo)

	// Зрушення за часом запитів метрик щодо запитів списку відео
	time.Sleep(*ShiftPeriodMetric)

	// Таймер для періодичної перевірки метрик відео
	timerMeter := time.Tick(*PeriodMeter)

	for {
		select {
		case <-timerChannel:
			Logger.Warn("timerChannel")
			//			go checkChannels()
		case <-timerVideo:
			Logger.Warn("timerVideo")
			//			go checkVideos()
		case <-timerMeter:
			Logger.Warn("timerMeter")
			//			go getMeters()
		case s := <-quit:
			switch s {

			// kill -SIGQUIT XXXX
			// ps aux | grep -i cmd | grep -v grep | awk {'print $2'} | xargs kill -3
			case syscall.SIGQUIT:
				printStatus()

			// kill -SIGINT XXXX or Ctrl+c
			// ps aux | grep -i cmd | grep -v grep | awk {'print $2'} | xargs kill -2
			case syscall.SIGINT, syscall.SIGTERM:
				Logger.Info("Server shutdown, wait 5 seconds")
				time.Sleep(5 * time.Second)
				closeDB()
				Logger.Info("Server shutdown")
				return

			default:
				Logger.Info("Unknown signal.")
			}
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}

}

func printStatus() {
	Logger.Infof("print status to %v", *FileStatus)

	timeFormatFull := "2006-01-02 15:04:05"
	timeFormatTime := "15:04:05"

	f, err := os.OpenFile(*FileStatus, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0660)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()

	fmt.Fprintf(f, "status on date: %v\n", time.Now().Format(timeFormatFull))

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	fmt.Fprintln(f, "Memory statistics:")
	fmt.Fprintf(f, "Alloc: \t\t%v\n", m.Alloc)
	fmt.Fprintf(f, "Total: \t\t%v\nAlloc", m.TotalAlloc)
	fmt.Fprintf(f, "Sys: \t%v\n", m.Sys)
	fmt.Fprintf(f, "Lookups: \t%v\n", m.Lookups)
	fmt.Fprintf(f, "Mallocs: \t%v\n", m.Mallocs)
	fmt.Fprintf(f, "Frees: \t\t%v\n", m.Frees)
	fmt.Fprintf(f, "HeapAlloc: \t%v\n", m.HeapAlloc)
	fmt.Fprintf(f, "HeapSys: \t%v\n", m.HeapSys)
	fmt.Fprintf(f, "HeapIdle: \t%v\n", m.HeapIdle)
	fmt.Fprintf(f, "HeapInuse: \t%v\n", m.HeapInuse)
	fmt.Fprintf(f, "HeapReleased: \t%v\n", m.HeapReleased)
	fmt.Fprintf(f, "HeapObjects: \t%v\n", m.HeapObjects)
	fmt.Fprintf(f, "StackInuse: \t%v\n", m.StackInuse)
	fmt.Fprintf(f, "StackSys: \t%v\n", m.StackSys)
	fmt.Fprintf(f, "MSpanInuse: \t%v\n", m.MSpanInuse)
	fmt.Fprintf(f, "MSpanSys: \t%v\n", m.MSpanSys)
	fmt.Fprintf(f, "MCacheInuse: \t%v\n", m.MCacheInuse)
	fmt.Fprintf(f, "MCacheSys: \t%v\n", m.MCacheSys)
	fmt.Fprintf(f, "BuckHashSys: \t%v\n", m.BuckHashSys)
	fmt.Fprintf(f, "GCSys: \t\t%v\n", m.GCSys)
	fmt.Fprintf(f, "OtherSys: \t%v\n", m.OtherSys)
	fmt.Fprintf(f, "NextGC: \t%v\n", m.NextGC)
	fmt.Fprintf(f, "LastGC: \t%v\n", m.LastGC)
	fmt.Fprintf(f, "PauseTotalNs: \t%v\n", m.PauseTotalNs)
	fmt.Fprintf(f, "NumGC: \t%v\n", m.NumGC)
	fmt.Fprintf(f, "NumForcedGC: \t%v\n", m.NumForcedGC)
	fmt.Fprintf(f, "GCCPUFraction: \t%v\n", m.GCCPUFraction)
	fmt.Fprintf(f, "EnableGC: \t%v\n", m.EnableGC)
	fmt.Fprintf(f, "DebugGC: \t%v\n", m.DebugGC)

	fmt.Fprintln(f, "\nDatabase statistics:")
	fmt.Fprintf(f, "database open connections: %v \n", db.Stats().OpenConnections)

	fmt.Fprintln(f, "\nChannel statistics:")
	for _, ch := range channels.Channels {
		fmt.Fprintf(f, "ch: %v, pl: %v, del: %v, timeDel: %v\n", ch.Id, ch.Idpl, ch.Deleted, ch.TimeDeleted.Format(timeFormatFull))

		for id, video := range ch.Videos {
			fmt.Fprintf(f, "  id: %v, at: %v, dur:%10v, like:%8v, dislike:%8v, comment:%8v, view:%8v, tcount: %v, del:%6v, tdel: %v, title: %v\n",
				id, video.PublishedAt.Format(timeFormatFull), video.Duration,
				video.LikeCount, video.DislikeCount, video.CommentCount, video.ViewCount,
				video.TimeCount.Format(timeFormatTime), video.Deleted, video.TimeDeleted.Format(timeFormatTime),
				video.Title)
		}
		fmt.Fprintln(f, "")
	}
	fmt.Fprintln(f, "================================================================================================")
}

// getClient uses a Context and Config to retrieve a Token
// then generate a Client. It returns the generated Client.
func getClient(ctx context.Context, config *oauth2.Config) *http.Client {
	cacheFile, err := tokenCacheFile()
	if err != nil {
		Logger.Fatalf("Unable to get path to cached credential file. %v", err)
	}
	tok, err := tokenFromFile(cacheFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(cacheFile, tok)
	}

	Logger.Debugf("token=%v", tok)
	return config.Client(ctx, tok)
}

// getTokenFromWeb uses Config to request a Token.
// It returns the retrieved Token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	Logger.Warnf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		Logger.Fatalf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(oauth2.NoContext, code)
	if err != nil {
		Logger.Fatalf("Unable to retrieve token from web %v", err)
	}
	return tok
}

// tokenCacheFile generates credential file path/filename.
// It returns the generated credential path/filename.
func tokenCacheFile() (string, error) {
	if _, err := os.Stat(*CredentialFile); err != nil {
		return "", err
	} else {
		return *CredentialFile, nil
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
	Logger.Warnf("Saving credential file to: %s\n", file)
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		Logger.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

// Заповнюємо список каналів та відео з БД
func initChannels() {
	Logger.Debug("init channels")
	channelsFromDB, err := GetChannelsWithVideoFromDB()
	if err != nil {
		Logger.Errorf("Error get channels from DB: ", err)
	}
	Logger.Debugf("channels from DB: %v", channelsFromDB)

	if len(channelsFromDB.Channels) > 0 {
		channels = channelsFromDB
	} else {
		channels = YoutubeChannels{Channels: make(map[string]*YoutubeChannel)}
	}
}

// Перевіряємо список каналів, чи додав адміністратор нові, чи видалив, чи деактивував, та корегуємо
func checkChannels() {
	Logger.Debug("check channel")

	// Отримуємо перечень діючих Channel-ів з БД на даний час
	ids, err := GetChannelsIDsFromDB()

	if err != nil {
		Logger.Errorf("Error get id's channels: ", err)
		return
	}

	Logger.Debugf("ids from db: %v", ids)

	if ids != nil && len(ids) > 0 {

		channels.Mux.Lock()
		defer channels.Mux.Unlock()

		// Перевіряємо список на видалення чи деактивування
		for id, ch := range channels.Channels {

			_, ok := ids[id]
			if ok == false { // підлягає припиненню обробки
				if ch.Deleted { // вже помічений на припинення обробки
					if time.Since(ch.TimeDeleted) > *PeriodDeleted { // перевіряємо, чи не час припиняти обробку
						channels.Delete(id) // видалення
						Logger.Infof("ch: %v, stop processing channel", id)
					}
				} else { // ще не помічений на припинення обробки
					channels.SetDeletedChannel(id) // помічаємо: підлягае припиненню обробки
					Logger.Debugf("ch: %v, set stop processing channel", id)
				}
			} else { // не підлягае видаленню
				if ch.Deleted { // але раніше підлягав
					channels.CanselDeletedChannel(id) // відміна видалення
					Logger.Debugf("ch: %v, cansel stop processing channel", id)
				}
			}

		}

		// Перевіряємо список на додавання нових каналів
		for id, _ := range ids {
			_, ok := channels.Channels[id]
			if ok == false {
				channels.Append(id) // додаемо новий Channel
				Logger.Infof("ch: %v, Append channel", id)
			}
		}

	}
}

// Отримати тимчасовий список каналів для роботи з сервісами Youtube. Цей тимчасовий список потрібен щоб не
// блокувати надовго роботу з основним списком, в якій можуть додати, або видалити канал
// Плейлисти помічені на видалення ігноруються
func getRequestChannel() (map[string]*YoutubeChannel, map[string]*YoutubeChannel) {
	requestChannel := make(map[string]*YoutubeChannel)  // Канали з плейлистами
	requestPlaylist := make(map[string]*YoutubeChannel) // Канали без плейлистів (ще не заповнениі з youtube)

	// блокування потрібно щоб гарантовано не почати обробляти Channel якій видалений, чи деактивований
	channels.Mux.Lock()
	defer channels.Mux.Unlock()
	for id, channel := range channels.Channels {
		if !channel.Deleted { // додаються тільки робочі плейлисти
			if channel.Idpl == "" {
				requestPlaylist[id] = channel // id плейлиста ще немає, сперше його потрібно заповнити
			} else {
				requestChannel[id] = channel //  id плейлиста є, можемо отримувати дані по відео
			}
		}
	}

	return requestChannel, requestPlaylist
}

// Перевіряємо список відео каналу, чи були додані нові, чи вичерпався термін збору статистики на старих
func checkVideos() {
	Logger.Debug("check videos start")

	requestChannel, requestPlaylist := getRequestChannel() // отримуємо список каналів для запросів
	Logger.Debugf("request channels: %v", requestChannel)

	// Якщо є канали з незаповненими id плейлистів, запрошуємо ці дані з сервісу youtube. Отримувати діні
	// по відео ми ще не взмозі
	if len(requestPlaylist) > 0 {
		go fillPlaylistFromYoutube(requestPlaylist)
	}

	// Якщо є канали з заповненими id плейлистів, на їх основі запрошуємо дані по відео
	if len(requestChannel) > 0 {

		if countRequestPlaylistItems >= *CountRequestPlaylistItems {
			Logger.Info("from service youtube.seach")
			countRequestPlaylistItems = 0
			for _, channel := range requestChannel {
				// Запрос відео через youtube.seach.list
				go getVideosFromYoutubeSearch(channel)
			}
		} else {
			Logger.Info("from service youtube.playlistitems")
			for _, channel := range requestChannel {
				// Запрос відео через youtube.playlistitems.list
				go getVideosFromYoutubePlaylistItems(channel)
			}
			countRequestPlaylistItems++
		}
	}

	Logger.Debug("check videos end")
}

// Перевіряємо список відео конкретного каналу, чи були додані нові, чи вичерпався термін збору
// статистики на старих. Для отримання списку відео викоритовується сервіс
// youtube.playlistitems.list
func getVideosFromYoutubePlaylistItems(channel *YoutubeChannel) {
	Logger.Debugf("ch: %v, request from youtube.playlistitems, count videos: %v", channel.Id, len(channel.Videos))

	// перевіряємо плейлист на застарілs відео яке вже не потрібно обробляти
	if len(channel.Videos) > 0 {
		checkElapsedVideos(channel)
	}

	call := service.PlaylistItems.List(PLAYLISTITEMS__LIST_PART)
	call = call.MaxResults(*MaxRequestVideos)
	call = call.PlaylistId(channel.Idpl)
	response, err := call.Do()
	if err != nil {
		Logger.Errorf("ch: %v, error get playlistitems: %v", channel.Id, err)
		return
	}

	// перевіряємо на появу нових відео
	for _, item := range response.Items {
		videoId := item.Snippet.ResourceId.VideoId
		Logger.Debugf("ch: %v, video: %v, is new?", channel.Id, videoId)

		video, ok := channel.Videos[videoId]
		if ok == false { // такого відео ще нема, пробуємо додати
			addVideo(channel, videoId, item.Snippet.Title, item.Snippet.Description,
				item.Snippet.PublishedAt, "")
		} else { // перевіримо відео на зміни даних
			checkUpdateVideo(channel.Id, videoId, item.Snippet.Title, "", video)
		}
	}
	Logger.Debugf("ch: %v, count videos: %v", channel.Id, len(channel.Videos))

}

// Перевіряємо список відео конкретного каналу, чи були додані нові, чи вичерпався термін збору статистики на старих
// для отримання списку відео викоритовується сервіс https://www.googleapis.com/youtube/v3
//
// https://www.googleapis.com/youtube/v3/search?part=snippet%2Cid&channelId=UCRzL8jf39oEWyrPnjmhBa2w&maxResults=25
// &order=date&publishedAfter=2019-01-30T19%3A08%3A26.000Z&type=video&key={YOUR_API_KEY}
//
func getVideosFromYoutubeSearch(channel *YoutubeChannel) {
	Logger.Debugf("ch: %v, request from youtube.search, count videos: %v", channel.Id, len(channel.Videos))

	// перевіряємо плейлист на застарілі відео яке вже не потрібно обробляти
	if len(channel.Videos) > 0 {
		checkElapsedVideos(channel)
	}

	// Отримуємо тільки ті відео, в яких ще не вичерпався термін збору метрик
	t := time.Now().Add(-*PeriodСollection)

	call := service.Search.List(SEARCH_LIST_PART)
	call = call.MaxResults(*MaxRequestVideos)
	call = call.Type("video")
	call = call.ChannelId(channel.Id)
	call = call.Order("date")
	call = call.PublishedAfter(t.Format(LAYOUT_ISO_8601))

	Logger.Debugf("ch: %v, PublishedAfter: %v", channel.Id, t)

	response, err := call.Do()
	if err != nil {
		Logger.Errorf("ch: %v, error get channels: %v", channel.Id, err)
		return
	}

	// перевіряємо на появу нових відео
	for _, item := range response.Items {
		videoId := item.Id.VideoId
		Logger.Debugf("ch: %v, video: %v, is new?", channel.Id, videoId)

		video, ok := channel.Videos[videoId]
		if ok == false { // такого відео ще нема, пробуємо додати
			addVideo(channel, videoId, item.Snippet.Title, item.Snippet.Description,
				item.Snippet.PublishedAt, item.Snippet.LiveBroadcastContent)
		} else { // перевіримо відео на зміни даних
			checkUpdateVideo(channel.Id, videoId, html.UnescapeString(item.Snippet.Title),
				item.Snippet.LiveBroadcastContent, video)
		}
	}
	Logger.Debugf("ch: %v, count videos: %v", channel.Id, len(channel.Videos))
}

// Перевірити деталі відео, та при необхідності виправити
func checkUpdateVideo(chId, videoId, title, alive string, video *YoutubeVideo) {
	isUpdate := false

	// Чи не потрібно занести дані по тривалості відео (наприклад закінчився стрим і появився його запис)
	if alive != LIVE_BROADCAST_CONTENT && video.Duration == 0 {
		videoDetails := getVideoDetails(chId, videoId)
		if videoDetails != nil {
			if d, err := duration.Parse(videoDetails.Duration); err == nil {
				video.Duration = d
				isUpdate = true
			} else {
				Logger.Error("ch: %v, video: %v, error parse duration: %v",
					chId, videoId, videoDetails.Duration)
			}
		}
	}

	// Відео змінило опис
	if video.Title != title {
		video.Title = title
		isUpdate = true
	}

	if isUpdate {
		err := UpdateVideoInDB(videoId, video)
		if err != nil {
			Logger.Error(err)
			return
		}
		Logger.Infof("ch: %v, video: %v, update -> title: %v, duration: %v",
			chId, videoId, video.Title, video.Duration)
	}

}

// Перевіряє чи не настав час (задається через PeriodСollection) припинити обробку якихось відео
// Спочатку відео помічаєтеся для видалення, а через заданий час (PeriodDeleted) видаляється остаточно
// Рознесення в часі помітки відео на видалення і само видалення гарантує коректну роботу потоків програми
func checkElapsedVideos(channel *YoutubeChannel) {
	channel.Mux.Lock()
	defer channel.Mux.Unlock()

	countDeleted := 0
	for id, video := range channel.Videos {

		if video.Deleted { // якщо відео призначене для видалення
			countDeleted++
			if time.Since(video.TimeDeleted) > *PeriodDeleted { // перевіряємо, чи не час видаляти
				channel.Delete(id) // видалення
				Logger.Infof("ch: %v, video: %v, stop processing", channel.Id, id)
			}
		} else { // відео ще не призначене для видалення
			// Перевірка чи не потрібно припинити обробку відео за часом
			if time.Since(video.PublishedAt) > *PeriodСollection {
				video.SetDeleted()
				Logger.Infof("ch: %v, video: %v, set stop processing", channel.Id, id)
			}

		}
	}
	Logger.Infof("ch: %v, count videos - all: %v, stopped: %v", channel.Id, len(channel.Videos),
		countDeleted)
}

// Отримати детальні дані по відео
func getVideoDetails(channelId, videoId string) *youtube.VideoContentDetails {
	call := service.Videos.List(VIDEOS_LIST_PART_DETAILS)
	call = call.Id(videoId)

	response, err := call.Do()
	if err != nil {
		Logger.Errorf("ch: %v, error get video details by ids=%v, error=%v", channelId, videoId, err)
		return nil
	}

	if len(response.Items) > 0 {
		return response.Items[0].ContentDetails
	}

	return nil
}

// додаємо відео для збору статистики
func addVideo(channel *YoutubeChannel, videoId, title, description, publishedAt, alive string) {
	channelId := channel.Id
	if videoId == "" {
		Logger.Errorf("ch: %v, error: video id is empty", channelId)
		return
	}

	// етап перевірки чи не застаріле відео
	timePublishedAt, err := time.Parse(LAYOUT_ISO_8601, publishedAt)
	if err != nil {
		Logger.Errorf("ch: %v, error parse PublishedAt %v", channels, publishedAt)
		return
	}
	timeElapsed := time.Since(timePublishedAt)
	if timeElapsed > *PeriodСollection {
		Logger.Debugf("ch: %v, video: %v, skip proccessing, time elapsed: %v", channelId, videoId, timeElapsed)
		return
	}

	var videoDuration time.Duration

	// Якщо це не пряма трансляція отримуємо тривалість відео
	if alive != LIVE_BROADCAST_CONTENT {
		videoDetails := getVideoDetails(channelId, videoId)
		if videoDetails != nil {
			if d, err := duration.Parse(videoDetails.Duration); err == nil {
				videoDuration = d
			}
		}
	}

	// відео пройшло перевірку, додаємо його для збору статистики
	err = AddVideoToDB(videoId, channelId, timePublishedAt, title, description, videoDuration)
	if err != nil {
		Logger.Error(err)
		return
	}

	channel.Mux.Lock()
	defer channel.Mux.Unlock()
	channel.Append(videoId, &YoutubeVideo{PublishedAt: timePublishedAt, Title: title, Deleted: false})
	Logger.Infof("ch: %v, video: %v, add new at: %v, title: %v, stream: %v", channelId, videoId, timePublishedAt, title, alive)
}

func getMeters() {
	Logger.Debug("check meters start")

	requestChannel, _ := getRequestChannel() // отримуємо список каналів для запросів
	Logger.Debugf("check meters, count request channels: %v", len(requestChannel))

	for _, channels := range requestChannel {
		go getMetersVideos(channels)
	}
	Logger.Debug("check meters end")
}

func getMetersVideos(channels *YoutubeChannel) {
	if len(channels.Videos) > 0 {
		mRrequestVideos := getRequestVideosFromChannel(channels)
		for i := 0; i < len(mRrequestVideos); i++ {
			getMetersVideosInd(channels.Id, mRrequestVideos[i])
		}
	} else {
		Logger.Infof("ch: %v, skip - count videos 0", channels.Id)
		return
	}
}

// Отримати тимчасовий список відео для роботи з сервісами Youtube. Цей тимчасовий список потрібен щоб не
// блокувати надовго роботу з основним списком, в якій можуть додати, або видалити відео
// Відео помічені на видалення ігноруються
// Список відео в запросі ділимо на частини згідно з дозволеною кількістью youtube api: зараз 50
// Повертаємо массив з частинами запросу кожна по 50 відео
func getRequestVideosFromChannel(channel *YoutubeChannel) []map[string]*YoutubeVideo {
	Logger.Debugf("ch: %v, get request channel, count video: all: %v", channel.Id, len(channel.Videos))
	mRequestVideos := []map[string]*YoutubeVideo{}

	var requestVideos map[string]*YoutubeVideo // перша частина запросу: перші 50 відео
	requestVideos = make(map[string]*YoutubeVideo)

	mRequestVideos = append(mRequestVideos, requestVideos)

	// блокування потрібно щоб гарантовано не почати обробляти відео які видалені, чи деактивовані
	channel.Mux.Lock()
	defer channel.Mux.Unlock()

	count := 0
	countall := 0
	for id, video := range channel.Videos {
		if !video.Deleted { // додаються тільки робочі відео
			// обробляємо тільки дозволену кількість відео. Запрос ділимо на частини
			// Робимо нову частину запросу: ще 50 відео
			if count >= *MaxRequestCountVideoID {
				requestVideos = make(map[string]*YoutubeVideo)
				mRequestVideos = append(mRequestVideos, requestVideos)
				count = 0
			}
			requestVideos[id] = video

			count++
			countall++
		}
	}
	Logger.Debugf("ch: %v, get request channel, count video: all: %v, request: %v", channel.Id, len(channel.Videos),
		countall)

	return mRequestVideos
}

func getMetersVideosInd(idch string, requestVideos map[string]*YoutubeVideo) {
	Logger.Debugf("ch: %v, getMetersVideo, count request videos: %v", idch, len(requestVideos))

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
	ids := bIds.String()

	call := service.Videos.List(VIDEOS_LIST_PART)
	call = call.Id(ids)

	response, err := call.Do()
	if err != nil {
		Logger.Errorf("ch: %v, error get video list by ids=%v, error=%v", idch, ids, err)
		return
	}

	var metrics = []*Metrics{}

	for _, item := range response.Items {
		videoId := item.Id

		rVideo, ok := requestVideos[videoId]
		if ok == true {

			// Відео видалене з каналу тому нема статистиці по ньому, тож припиняємо його обробку
			if item == nil || item.Statistics == nil {
				rVideo.SetDeleted()
				Logger.Infof("ch: %v, video: %v set deleted because statistics is null", idch, videoId)
				continue
			}
			videoCommentCount := item.Statistics.CommentCount
			videoLikeCount := item.Statistics.LikeCount
			videoDislikeCount := item.Statistics.DislikeCount
			videoViewCount := item.Statistics.ViewCount

			Logger.Debugf("ch: %v, video: %v, comment: %5v, like: %6v, dislike: %6v, view: %8v",
				idch,
				videoId,
				videoCommentCount,
				videoLikeCount,
				videoDislikeCount,
				videoViewCount)

			// Заносимо метрики до БД в двох випадках:
			//   1. якщо пройшов заданий період ( PeriodCount )
			//   2. якщо змінилась будь яка метрика (лайки, дізлайки тощо)
			if time.Since(rVideo.TimeCount) > *PeriodCount ||
				rVideo.CommentCount != videoCommentCount ||
				rVideo.LikeCount != videoLikeCount ||
				rVideo.DislikeCount != videoDislikeCount ||
				rVideo.ViewCount != videoViewCount {

				rVideo.SetMetrics(videoCommentCount, videoLikeCount, videoDislikeCount, videoViewCount)
				metrics = append(metrics, &Metrics{videoId, videoCommentCount, videoLikeCount,
					videoDislikeCount, videoViewCount, time.Now()})
				Logger.Debugf("ch: %v, video: %v, save metrics", idch, videoId)
			}
		} else {
			Logger.Errorf("ch: %v, Cannot get request video with id %v=", idch, videoId)
		}
	}

	if len(metrics) > 0 {
		AddMetricToDB(metrics)
	}

	Logger.Infof("ch: %v, video's metrics - save: %v, skip %v", idch, len(metrics),
		len(requestVideos)-len(metrics))
}

// Заповнюємо дані по id плейлисту uploads
func fillPlaylistFromYoutube(channels map[string]*YoutubeChannel) {
	Logger.Debugf("fill playlist by channel ids")

	mIds := getSliceIds(channels)

	for i := 0; i < len(mIds); i++ {
		items := getPlailistIDsFromYoutube(mIds[i].String())
		for _, item := range items {
			Logger.Debugf("get item %v", item)

			channelId := item.Id
			channel, ok := channels[channelId]
			if ok {
				channel.Idpl = item.ContentDetails.RelatedPlaylists.Uploads
				Logger.Infof("ch: %v, set pl: %v", channelId, channel.Idpl)
			} else {
				Logger.Errorf("ch: %v, error set pl: %v", channelId, channel.Idpl)
			}
		}
	}
}

// формуємо массив з id каналів вигляду: id,id,id максимальна кількість MaxRequestCountChannelID
func getSliceIds(channels map[string]*YoutubeChannel) []*bytes.Buffer {
	bIds := bytes.NewBuffer([]byte(""))
	var isFirst = true
	mIds := []*bytes.Buffer{}
	mIds = append(mIds, bIds)
	count := 0
	for key, _ := range channels {
		if count >= *MaxRequestCountChannelID {
			bIds = bytes.NewBuffer([]byte(""))
			mIds = append(mIds, bIds)

			count = 0
			isFirst = true
		}

		if isFirst {
			isFirst = false
		} else {
			bIds.WriteString(",")
		}
		bIds.WriteString(key)

		count++
	}

	return mIds
}

func getPlailistIDsFromYoutube(ids string) []*youtube.Channel {
	Logger.Debugf("get playlist by channel ids: %v", ids)

	call := service.Channels.List(CHANNELS_LIST_PART)
	call = call.Id(ids)

	response, err := call.Do()
	if err != nil {
		Logger.Errorf("error get channels.list ids=%v, error=%v", ids, err)
		return nil
	}

	return response.Items
}

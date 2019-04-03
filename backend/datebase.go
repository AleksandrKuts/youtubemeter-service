package backend

import (
	"database/sql"
	"encoding/json"
	"errors"
	_ "github.com/lib/pq"
	"strconv"
	"strings"
	"time"
)

// The layout defines the format by showing how the reference time, defined to be.
// timestamp with time zone;
const TIME_LAYOUT = "2006-01-02T15:04:05.999999-07:00"

const INSERT_CHANNEL = "INSERT INTO channel ( id, title, enable ) VALUES ( $1, $2, $3 )"
const UPDATE_CHANNEL = "UPDATE channel SET title=$2, enable=$3 WHERE id = $1"
const DELETE_CHANNEL = "DELETE FROM channel WHERE id = $1"
const GET_CHANNELS = "SELECT id, TRIM(title), enable, timeadd, countvideo FROM channel ORDER BY title"
const GET_CHANNELS_ENABLE = "SELECT id, TRIM(title), enable, timeadd, countvideo FROM channel " +
	"WHERE enable = true ORDER BY title"
const GET_METRICS_BY_IDVIDEO = "Select * FROM return_metrics($1)"
const GET_METRICS_BY_IDVIDEO_BETWEEN_DATE = "Select * FROM return_metrics($1, $2, $3)"

const GET_VIDEO_BY_ID = "SELECT * FROM return_video($1)"

const GET_VIDEOS= "SELECT v.id, TRIM(v.title), v.publishedat, TRIM(ch.title) as ptitle, v.duration, v.idch FROM video v" +
	" LEFT JOIN channel ch ON ch.id = v.idch" +
	" WHERE ch.enable = true ORDER BY publishedat DESC LIMIT $1 OFFSET $2"
	
const GET_VIDEOS_BY_ID_CHANNEL = "SELECT id, TRIM(title), publishedat, '' ptitle, v.duration, v.idch FROM video v" + 
	" WHERE idch = $1 " +
	" ORDER BY publishedat DESC LIMIT $2 OFFSET $3"
	
const GET_GLOBAL_COUNTS = "select count(*) as count, SUM(countvideo) as countvideo FROM channel WHERE enable = TRUE"	

const NO_DATA = "No data"

// creat connections string
// example: host=127.0.0.100 port=5432 dbname=base1 user=user1 password=lalala sslmode=disable"
var connStrForDatabse string

var db *sql.DB
var errDB error

// формуємо підключення до Бази Даних (БД), підключення відкрита на протязі всієї роботи програми
func init() {
	// creat connections string
	// example: host=127.0.0.100 port=5432 dbname=base1 user=user1 password=lalala sslmode=disable"
	connStrForDatabse = "host=" + *DBHost +
		" port=" + *DBPort +
		" dbname=" + *DBName +
		" user=" + *DBUser +
		" password=" + *DBPassword +
		" sslmode=" + *DBSSLMode

	Logger.Debugf("connStr=%s", connStrForDatabse)

	db, errDB = sql.Open("postgres", connStrForDatabse)
	if errDB != nil {
		Logger.Errorf("error open database: %v", errDB)
	}

	// перевірка доступності
	err := db.Ping()
	if err != nil { // далі робити не можна
		Logger.Fatalf("error ping database: %v", err)
	}

}

// коректне закриття з'єднання з БД
func closeDB() {
	Logger.Infof("close database with %v open connections", db.Stats().OpenConnections)

	err := db.Close()
	if err != nil {
		Logger.Errorf("error close database: %v", err)
	}
}

// Додати канал до БД
func addChannelDB(channel *Channel) error {
	res, err := db.Exec(INSERT_CHANNEL, channel.Id, channel.Title, channel.Enable)
	if err != nil {
		Logger.Errorf("err=%v", err)
		return err
	}

	_, err = res.RowsAffected()
	if err != nil {
		Logger.Errorf("err=%v", err)
		return err
	} else {
		Logger.Debugf("insert channel: id=%v, title=%v, enable=%v", channel.Id, channel.Title, channel.Enable)
	}

	return nil
}

// Оновити канал в БД
func updateChannelDB(id string, channel *Channel) error {
	res, err := db.Exec(UPDATE_CHANNEL, id, channel.Title, channel.Enable)
	if err != nil {
		Logger.Errorf("err=%v", err)
		return err
	}

	_, err = res.RowsAffected()
	if err != nil {
		Logger.Errorf("err=%v", err)
		return err
	} else {
		Logger.Debugf("update channel: id=%v, title=%v, enable=%v", id, channel.Title, channel.Enable)
	}

	return nil
}

// Видалити канал з БД
func deleteChannelDB(channelId string) error {
	res, err := db.Exec(DELETE_CHANNEL, channelId)
	if err != nil {
		Logger.Errorf("err=%v", err)
		return err
	}

	_, err = res.RowsAffected()
	if err != nil {
		Logger.Errorf("err=%v", err)
		return err
	} else {
		Logger.Debugf("deleted channel: id=%v", channelId)
	}

	return nil
}

// Отримати канали
// onlyEnable - які канали вибирати
//   true  - тільки активні, якщо канал не активний його треба активувати через інтерфейс адміністратора
//   false - всі
func getChannelsFromDB(onlyEnable bool) ([]Channel, error) {
	Logger.Debugf("dbstats=%v", db.Stats())

	var rows *sql.Rows
	var err error

	if onlyEnable {
		rows, err = db.Query(GET_CHANNELS_ENABLE)		
	} else {
		rows, err = db.Query(GET_CHANNELS)		
	}
	
	if err != nil {
		Logger.Errorf("Error get channels: %v", err)
		return nil, err
	}

	if rows == nil {
		return nil, errors.New( NO_DATA )
	}

	defer rows.Close()

	response := []Channel{}

	for rows.Next() {
		var Id string
		var Title string
		var Enable bool
		var Timeadd time.Time
		var countvideo int

		rows.Scan(&Id, &Title, &Enable, &Timeadd, &countvideo)
		Id = strings.TrimSpace(Id)
		Title = strings.TrimSpace(Title)

		response = append(response, Channel{Id, Title, Enable, Timeadd, countvideo})
	}
	err = rows.Err()
	if err != nil {
		Logger.Error(err)
		return nil, err
	}

	Logger.Debugf("Success get channels from DB")

	return response, nil
}

// Отримати метрики по відео id за заданий період
func getMetricsByIdFromDB(id string, from, to string) ([]*Metrics, error) {
	var rows *sql.Rows
	var err error

	Logger.Debugf("id: %v, from: %v, to: %v", id, from, to)

	// якщо період не заданий обираємо всі дані
	if from == "" && to == "" {
		rows, err = db.Query(GET_METRICS_BY_IDVIDEO, id)
	} else { // обраний період
		/* перевіряємо та форматуємо дату з якої вибираємо */
		sFrom, err := checkDate(from)
		if err != nil {
			return nil, err
		}

		/* перевіряємо та форматуємо дату по яку вибираємо */
		sTo, err := checkDate(to)
		if err != nil {
			return nil, err
		}

		rows, err = db.Query(GET_METRICS_BY_IDVIDEO_BETWEEN_DATE, id, sFrom, sTo)
	}

	if err != nil {
		Logger.Errorf("Error get metrics: %v", err)
		return nil, err
	}

	if rows == nil {
		return nil, errors.New( NO_DATA )
	}

	defer rows.Close()
	
	response := []*Metrics{}

	for rows.Next() {
		var commentCount uint64
		var likeCount uint64
		var dislikeCount uint64
		var viewCount uint64
		var vTime time.Time

		rows.Scan(&commentCount, &likeCount, &dislikeCount, &viewCount, &vTime)

		response = append(response, &Metrics{commentCount, likeCount, dislikeCount, viewCount, vTime})
	}
	err = rows.Err()
	if err != nil {
		Logger.Error(err)
		return nil, err
	}

	return response, nil
}

// Отримати опис відео по його id
func getVideoByIdFromDB(id string) ( *YoutubeVideo, error) {
	if id == "" {
		return nil, errors.New("video id is null")
	}

	var title string
	var description string
	var idch string
	var chtitle string
	var publishedat time.Time
	var count_metrics int
	var max_timemetric time.Time
	var min_timemetric time.Time
	var duration time.Duration

	err := db.QueryRow(GET_VIDEO_BY_ID, id).Scan(&title, &description, &idch, &chtitle, &publishedat, 
				&count_metrics, &min_timemetric, &max_timemetric, &duration )
	if err != nil {
		Logger.Errorf("Error get videos by id: %v", err)
		return nil, err
	}
	
	youtubeVideo := &YoutubeVideo{strings.TrimSpace(title), strings.TrimSpace(description), strings.TrimSpace(idch), 
			strings.TrimSpace(chtitle), publishedat, count_metrics, max_timemetric, min_timemetric, duration}
	
	Logger.Debugf("id: %v, title: %v, description: %v, idch: %v, chtitle: %v, publishedat: %v, count_metrics: %v, max_timemetric: %v, min_timemetric: %v, duration: %v", 
			id, title, description, idch, chtitle, publishedat, count_metrics, max_timemetric, min_timemetric, duration)

	return youtubeVideo, nil
}

// Отримати список відео по id каналу
func getVideosByChannelIdFromDB(id string, offset int) ([]byte, error) {
	Logger.Debugf("id: %v, offset: %v", id, offset)

	var rows *sql.Rows
	var err error

	if id == "" {
		rows, err = db.Query(GET_VIDEOS, *MaxViewVideosInChannel, offset)
	} else {
		rows, err = db.Query(GET_VIDEOS_BY_ID_CHANNEL, id, *MaxViewVideosInChannel, offset)
	}

	if err != nil {
		Logger.Errorf("Error get videos by plailist id: %v", err)
		return nil, err
	}

	if rows == nil {
		return nil, errors.New( NO_DATA )
	}

	defer rows.Close()

	response := []YoutubeVideoShort{}

	for rows.Next() {
		var id string
		var title string
		var publishedat time.Time
		var ptitle string
		var duration time.Duration
		var chid string

		rows.Scan(&id, &title, &publishedat, &ptitle, &duration, &chid)

		response = append(response, YoutubeVideoShort{id, title, publishedat, ptitle, duration, chid})
	}
	err = rows.Err()
	if err != nil {
		Logger.Error(err)
		return nil, err
	}

	// Конвертуємо відповідь в json-формат
	stringVideos, err := json.Marshal(response)

	if err != nil {
		Logger.Errorf("Error convert select to YoutubeVideoShort: response=%v, error=%v", response, err)
		return nil, err
	}

	Logger.Debugf("id: %v, channel: %v", id, string(stringVideos))

	return stringVideos, nil
}

// Отримати опис відео по його id
func getGlobalCountsFromDB(version string) ( *GlobalCounts, error) {
	var countChannels int = 0
	var countVideos int = 0

	err := db.QueryRow(GET_GLOBAL_COUNTS).Scan(&countChannels, &countVideos)
	if err != nil {
		Logger.Errorf("Error get global counts from DB: %v", err)
	}
	
	globalCounts := &GlobalCounts{CountChannels: countChannels, CountVideos: countVideos, TimeUpdate: time.Now(), 
		MaxVideoCount: *MaxViewVideosInChannel, PeriodVideoCache: *PeriodVideoCache / 1000000,
		Version: version, ListenAdmin: *ListenAdmin}
	
	Logger.Debugf("countChannels: %v, countVideos: %v", countChannels, countVideos)

	return globalCounts, nil
}



// Перевірка дати, заданої рядком мілісекунд, та її форматування
// якщо дата не задана (пустий рядок), повертаємо пустий рядок
// якщо задана, перевіряємо коректність та форматуємо в timestamp with time zone згідно TIME_LAYOUT
func checkDate(sdt string) (string, error) {
	if sdt == "" {
		return "", nil
	} else {
		millis, err := strconv.ParseInt(sdt, 10, 64)
		if err != nil {
			Logger.Errorf("Error convert string date %v to timestamp", sdt)
			return "", err
		}
		return time.Unix(0, millis*int64(time.Millisecond)).Format(TIME_LAYOUT), nil
	}
}

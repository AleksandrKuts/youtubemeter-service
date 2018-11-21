package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	_ "github.com/lib/pq"
	"github.com/AleksandrKuts/youtubemeter-service/backend/config"
	"strconv"
	"strings"
	"time"
)

// The layout defines the format by showing how the reference time, defined to be.
// timestamp with time zone;
const TIME_LAYOUT = "2006-01-02T15:04:05.999999-07:00"

const INSERT_PLAYLIST = "INSERT INTO playlist ( id, title, enable, idch ) VALUES ( $1, $2, $3, $4)"
const UPDATE_PLAYLIST = "UPDATE playlist SET title=$2, enable=$3, idch=$4 WHERE id = $1"
const DELETE_PLAYLIST = "DELETE FROM playlist WHERE id = $1"
const GET_PLAYLISTS = "SELECT id, TRIM(title), enable, idch, timeadd, countvideo FROM playlist ORDER BY title"
const GET_PLAYLISTS_ENABLE = "SELECT id, TRIM(title), enable, idch, timeadd, countvideo FROM playlist " +
	"WHERE enable = true ORDER BY title"
const GET_METRICS_BY_IDVIDEO = "Select * FROM return_metrics($1)"
const GET_METRICS_BY_IDVIDEO_BETWEEN_DATE = "Select * FROM return_metrics($1, $2, $3)"

const GET_VIDEO_BY_ID = "SELECT * FROM return_video($1)"

const GET_VIDEOS= "SELECT v.id, TRIM(v.title), v.publishedat, TRIM(p.title) as ptitle FROM video v" +
	" LEFT JOIN playlist p ON p.id = v.idpl" +
	" WHERE p.enable = true ORDER BY publishedat DESC LIMIT $1 OFFSET $2"
	
const GET_VIDEOS_BY_ID_PLAYLIST = "SELECT id, TRIM(title), publishedat, '' ptitle FROM video v" + 
	" WHERE idpl = $1 " +
	" ORDER BY publishedat DESC LIMIT $2 OFFSET $3"
	
const GET_GLOBAL_COUNTS = "select count(*) as count, SUM(countvideo) as countvideo FROM playlist WHERE enable = TRUE"	

// creat connections string
// example: host=127.0.0.100 port=5432 dbname=base1 user=user1 password=lalala sslmode=disable"
var connStrForDatabse string

var db *sql.DB
var errDB error

// формуємо підключення до Бази Даних (БД), підключення відкрита на протязі всієї роботи програми
func init() {
	// creat connections string
	// example: host=127.0.0.100 port=5432 dbname=base1 user=user1 password=lalala sslmode=disable"
	connStrForDatabse = "host=" + *config.DBHost +
		" port=" + *config.DBPort +
		" dbname=" + *config.DBName +
		" user=" + *config.DBUser +
		" password=" + *config.DBPassword +
		" sslmode=" + *config.DBSSLMode

	log.Debugf("connStr=%s", connStrForDatabse)

	db, errDB = sql.Open("postgres", connStrForDatabse)
	if errDB != nil {
		log.Errorf("error open database: %v", errDB)
	}

	// перевірка доступності
	err := db.Ping()
	if err != nil { // далі робити не можна
		log.Fatalf("error ping database: %v", err)
	}

}

// коректне закриття з'єднання з БД
func closeDB() {
	log.Infof("close database with %v open connections", db.Stats().OpenConnections)

	err := db.Close()
	if err != nil {
		log.Errorf("error close database: %v", err)
	}
}

// Додати плей-лист до БД
func addPlayListDB(playlist *PlayList) error {
	res, err := db.Exec(INSERT_PLAYLIST, playlist.Id, playlist.Title, playlist.Enable, playlist.Idch)
	if err != nil {
		log.Errorf("err=%v", err)
		return err
	}

	_, err = res.RowsAffected()
	if err != nil {
		log.Errorf("err=%v", err)
		return err
	} else {
		log.Debugf("insert playlist: id=%v, title=%v, enable=%v, idch=%v", playlist.Id, playlist.Title, playlist.Enable, playlist.Idch)
	}

	return nil
}

// Оновити плей-лист в БД
func updatePlayListDB(id string, playlist *PlayList) error {
	res, err := db.Exec(UPDATE_PLAYLIST, id, playlist.Title, playlist.Enable, playlist.Idch)
	if err != nil {
		log.Errorf("err=%v", err)
		return err
	}

	_, err = res.RowsAffected()
	if err != nil {
		log.Errorf("err=%v", err)
		return err
	} else {
		log.Debugf("update playlist: id=%v, title=%v, enable=%v, idch=%v", id, playlist.Title, playlist.Enable, playlist.Idch)
	}

	return nil
}

// Видалити плей-лист з БД
func deletePlayListDB(playlistId string) error {
	res, err := db.Exec(DELETE_PLAYLIST, playlistId)
	if err != nil {
		log.Errorf("err=%v", err)
		return err
	}

	_, err = res.RowsAffected()
	if err != nil {
		log.Errorf("err=%v", err)
		return err
	} else {
		log.Debugf("deleted playlist: id=%v", playlistId)
	}

	return nil
}

// Отримати плейлисти
// onlyEnable - які плейлисти вибирати
//   true  - тільки активні, якщо плейлист не активний його треба активувати через інтерфейс адміністратора
//   false - всі
func getPlaylistsFromDB(onlyEnable bool) ([]PlayList, error) {
	log.Debugf("dbstats=%v", db.Stats())

	var rows *sql.Rows
	var err error

	if onlyEnable {
		rows, err = db.Query(GET_PLAYLISTS_ENABLE)		
	} else {
		rows, err = db.Query(GET_PLAYLISTS)		
	}
	
	if err != nil {
		log.Errorf("Error get playlists: %v", err)
		return nil, err
	}
	defer rows.Close()

	response := []PlayList{}

	for rows.Next() {
		var Id string
		var Title string
		var Enable bool
		var Idch string
		var Timeadd time.Time
		var countvideo int

		rows.Scan(&Id, &Title, &Enable, &Idch, &Timeadd, &countvideo)
		Id = strings.TrimSpace(Id)
		Title = strings.TrimSpace(Title)
		Idch = strings.TrimSpace(Idch)

		response = append(response, PlayList{Id, Title, Enable, Idch, Timeadd, countvideo})
	}
	err = rows.Err()
	if err != nil {
		log.Error(err)
		return nil, err
	}

	log.Error("Success get playlists from DB")

	return response, nil
}

// Отримати метрики по відео id за заданий період
func getMetricsByIdFromDB(id string, from, to string) ([]*Metrics, error) {
	var rows *sql.Rows
	var err error

	log.Debugf("id: %v, from: %v, to: %v", id, from, to)

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
		log.Errorf("Error get metrics: %v", err)
		return nil, err
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
		log.Error(err)
		return nil, err
	}

	return response, nil
}

// Отримати опис відео по його id
func getVideoByIdFromDB(id string) ( *YoutubeVideo, error) {
	if id == "" {
		return nil, errors.New("video id is null")
	}

	var idpl string
	var title string
	var description string
	var chtitle string
	var chid string
	var publishedat time.Time
	var count_metrics int
	var max_timemetric time.Time
	var min_timemetric time.Time

	err := db.QueryRow(GET_VIDEO_BY_ID, id).Scan(&idpl, &title, &description, &chtitle, &chid, &publishedat, &count_metrics, &max_timemetric, &min_timemetric)
	if err != nil {
		log.Errorf("Error get videos by id: %v", err)
		return nil, err
	}
	
	youtubeVideo := &YoutubeVideo{strings.TrimSpace(idpl), strings.TrimSpace(title), strings.TrimSpace(description), 
			strings.TrimSpace(chtitle), strings.TrimSpace(chid), publishedat, count_metrics, max_timemetric, min_timemetric}
	
	log.Debugf("id: %v, idpl: %v, title: %v, description: %v, chtitle: %v, chid: %v, publishedat: %v, count_metrics: %v, max_timemetric: %v, min_timemetric: %v", 
			id, idpl, title, description, chtitle, chid, publishedat, count_metrics, max_timemetric, min_timemetric)

	return youtubeVideo, nil
}

// Отримати список відео по id плейлиста
func getVideosByPlayListIdFromDB(id string, offset int) ([]byte, error) {
	log.Debugf("id: %v, offset: %v", id, offset)

	var rows *sql.Rows
	var err error

	if id == "" {
		rows, err = db.Query(GET_VIDEOS, *config.MaxViewVideosInPlayLists, offset)
	} else {
		rows, err = db.Query(GET_VIDEOS_BY_ID_PLAYLIST, id, *config.MaxViewVideosInPlayLists, offset)
	}

	if err != nil {
		log.Errorf("Error get videos by plailist id: %v", err)
		return nil, err
	}
	defer rows.Close()

	response := []YoutubeVideoShort{}

	for rows.Next() {
		var id string
		var title string
		var publishedat time.Time
		var ptitle string

		rows.Scan(&id, &title, &publishedat, &ptitle)

		response = append(response, YoutubeVideoShort{id, title, publishedat, ptitle})
	}
	err = rows.Err()
	if err != nil {
		log.Error(err)
		return nil, err
	}

	// Конвертуємо відповідь в json-формат
	stringVideos, err := json.Marshal(response)

	if err != nil {
		log.Errorf("Error convert select to YoutubeVideoShort: response=%v, error=%v", response, err)
		return nil, err
	}

	log.Debugf("id: %v, playlist: %v", id, string(stringVideos))

	return stringVideos, nil
}

// Отримати опис відео по його id
func getGlobalCountsFromDB() ( *GlobalCounts, error) {
	var countPlaylists int
	var countVideos int

	err := db.QueryRow(GET_GLOBAL_COUNTS).Scan(&countPlaylists, &countVideos)
	if err != nil {
		log.Errorf("Error get global counts: %v", err)
		return nil, err
	}
	
	globalCounts := &GlobalCounts{CountPlaylists: countPlaylists, CountVideos: countVideos, TimeUpdate: time.Now(), 
		MaxVideoCount: *config.MaxViewVideosInPlayLists, PeriodVideoCache: *config.PeriodVideoCache / 1000000}
	
	log.Debugf("countPlaylists: %v, countVideos: %v", countPlaylists, countVideos)

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
			log.Errorf("Error convert string date %v to timestamp", sdt)
			return "", err
		}
		return time.Unix(0, millis*int64(time.Millisecond)).Format(TIME_LAYOUT), nil
	}
}

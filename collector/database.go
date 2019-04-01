package collector

import (
	"database/sql"
	"errors"
	"github.com/lib/pq"
	"strings"
	"time"
)

// The layout defines the format by showing how the reference time, defined to be.
// timestamp with time zone;
const TIME_LAYOUT = "2006-01-02T15:04:05.999999-07:00"

const GET_CHANNELS = "SELECT id FROM channel ch WHERE enable = true"

const GET_CHANNELS_WITH_VIDEO = "SELECT ch.id, v.id as vid, v.publishedat, TRIM(v.title) " +
	"FROM channel ch " +
	"LEFT JOIN video v ON v.idch = ch.id AND v.publishedat > $1 " +
	"WHERE ch.enable = true " +
	"ORDER BY ch.id"

const INSERT_VIDEO = "INSERT INTO video ( id, idch, publishedat, title, description ) " +
	"VALUES ( $1, $2, $3, $4, $5 ) " +
	"ON CONFLICT (id) DO UPDATE SET " +
	"publishedat = EXCLUDED.publishedat, title = EXCLUDED.title, description = EXCLUDED.description"

const UPDATE_VIDEO = "UPDATE video SET title = $1 WHERE id = $2"

const INSERT_METRICS = "INSERT INTO metric ( idVideo, CommentCount, LikeCount, DislikeCount, ViewCount ) " +
	"VALUES ( $1, $2, $3, $4, $5 )"

var db *sql.DB
var errDB error

func init() {
	// creat connections string
	// example: host=127.0.0.100 port=5432 dbname=base1 user=user1 password=lalala sslmode=disable"
	connStrForDatabse := "host=" + *DBHost +
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

	err := db.Ping()
	if err != nil {
		Logger.Fatalf("error ping database: %v", err)
	}

	Logger.Infof("open database with %v open connections", db.Stats().OpenConnections)
}

func closeDB() {
	Logger.Infof("close database with %v open connections", db.Stats().OpenConnections)

	err := db.Close()
	if err != nil {
		Logger.Errorf("error close database: %v", err)
	}
}

// Отримати массив ID списків відтворення та відео з БД 
func GetChannelsWithVideoFromDB() (YoutubeChannels, error) {
	Logger.Debugf("dbstats=%v", db.Stats())
	maxTimePublished := time.Now().Add(-*PeriodСollection)
	Logger.Debugf("get channels with videos, maxTimePublished: %v", maxTimePublished)

	var channels YoutubeChannels = YoutubeChannels{Channels: make(map[string] * YoutubeChannel)}

	rows, err := db.Query(GET_CHANNELS_WITH_VIDEO, maxTimePublished)
	if err != nil {
		Logger.Errorf("Error get channels: %v", err)
		return channels, err
	}
	defer rows.Close()

	ch := ""
	for rows.Next() {
		var id string
		var videoId string
		var publishedat time.Time
		var title string

		rows.Scan(&id, &videoId, &publishedat, &title)
		Logger.Debugf("ch: %v, video: %v, publishedat: %v, title: %v", id, videoId, publishedat, title)

		if ch != id {
			channels.Append(id)
			ch = id
		}
		if videoId != "" {
			channels.Channels[id].Append(videoId, &YoutubeVideo{PublishedAt: publishedat, 
					Deleted: false, Title: title})
		}
	}
	err = rows.Err()
	if err != nil {
		Logger.Error(err)
		return channels, err
	}

	return channels, nil
}

// Отримати массив ID списків відтворення
func GetChannelsIDsFromDB() (map[string]bool, error) {
	Logger.Debugf("dbstats=%v", db.Stats())

	rows, err := db.Query(GET_CHANNELS)
	if err != nil {
		Logger.Errorf("Error get channels: %v", err)
		return nil, err
	}
	defer rows.Close()

	response := make(map[string]bool)

	for rows.Next() {
		var Id string

		rows.Scan(&Id)
		Id = strings.TrimSpace(Id)

		response[Id] = true
	}
	err = rows.Err()
	if err != nil {
		Logger.Error(err)
		return nil, err
	}

	return response, nil
}

// Додати відео
func AddVideoToDB(id, idch string, publishedat time.Time, title, description string) error {
	if id == "" {
		return errors.New("Error add video, id is null")
	}
	if idch == "" {
		return errors.New("Error add video, idpl is null")
	}

	res, err := db.Exec(INSERT_VIDEO, id, idch, publishedat, title, description)
	if err != nil {
		Logger.Errorf("err=%v", err)
		return err
	}

	_, err = res.RowsAffected()
	if err != nil {
		Logger.Errorf("err=%v", err)
		return err
	} else {
		Logger.Debugf("insert video: id=%v, idpl=%v, publishedat=%v, title=%v", id, idch,
			publishedat, title)
	}

	return nil
}

// Оновити опис відео
func UpdateVideoInDB(id, title string) error {
	if id == "" {
		return errors.New("Error update video, id is null")
	}

	res, err := db.Exec(UPDATE_VIDEO, title, id)
	if err != nil {
		Logger.Errorf("err=%v", err)
		return err
	}

	_, err = res.RowsAffected()
	if err != nil {
		Logger.Errorf("err=%v", err)
		return err
	} else {
		Logger.Debugf("update video: id=%v, title=%v", id, title)
	}

	return nil
}


// Додати метрики
func AddMetricToDB(metrics []*Metrics) error {

	txn, err := db.Begin()
	if err != nil {
		Logger.Errorf("err=%v", err)
		return err
	}

	stmt, err := txn.Prepare(pq.CopyIn("metric", "idvideo", "commentcount", "likecount", "dislikecount", "viewcount",
		"timemetric"))
	if err != nil {
		Logger.Errorf("err=%v", err)
		return err
	}

	for _, metric := range metrics {
		_, err = stmt.Exec(metric.Id, metric.CommentCount, metric.LikeCount, metric.DislikeCount, metric.ViewCount,
			metric.Time)
		if err != nil {
			Logger.Errorf("err=%v", err)
			return err
		}
	}

	_, err = stmt.Exec()
	if err != nil {
		Logger.Errorf("err=%v", err)
		return err
	}

	err = stmt.Close()
	if err != nil {
		Logger.Errorf("err=%v", err)
		return err
	}

	err = txn.Commit()
	if err != nil {
		Logger.Errorf("err=%v", err)
		return err
	}

	return nil
}

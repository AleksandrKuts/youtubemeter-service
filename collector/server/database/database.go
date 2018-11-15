package database

import (
	"database/sql"
	"errors"
	"github.com/AleksandrKuts/youtubemeter-service/collector/config"
	"github.com/AleksandrKuts/youtubemeter-service/collector/server/model"
	"github.com/lib/pq"
	"go.uber.org/zap"
	"strings"
	"time"
)

// The layout defines the format by showing how the reference time, defined to be.
// timestamp with time zone;
const TIME_LAYOUT = "2006-01-02T15:04:05.999999-07:00"

const GET_PLAYLISTS = "SELECT pl.id FROM playlist pl WHERE pl.enable = true"

const GET_PLAYLISTS_WITH_VIDEO = "SELECT pl.id, v.id as vid, v.publishedat, v.title " +
	"FROM playlist pl " +
	"LEFT JOIN video v ON v.idpl = pl.id AND v.publishedat > $1 " +
	"WHERE pl.enable = true " +
	"ORDER BY pl.id"

const INSERT_VIDEO = "INSERT INTO video ( id, idpl, publishedat, title, description, chid, chtitle ) " +
	"VALUES ( $1, $2, $3, $4, $5, $6, $7 ) " +
	"ON CONFLICT (id) DO UPDATE SET " +
	"publishedat = EXCLUDED.publishedat, title = EXCLUDED.title, description = EXCLUDED.description, " +
	"chid = EXCLUDED.chid, chtitle = EXCLUDED.chtitle"

const INSERT_METRICS = "INSERT INTO metric ( idVideo, CommentCount, LikeCount, DislikeCount, ViewCount ) " +
	"VALUES ( $1, $2, $3, $4, $5 )"

var db *sql.DB
var errDB error
var log *zap.SugaredLogger

func init() {
	log = config.Logger

	// creat connections string
	// example: host=127.0.0.100 port=5432 dbname=base1 user=user1 password=lalala sslmode=disable"
	connStrForDatabse := "host=" + *config.DBHost +
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

	err := db.Ping()
	if err != nil {
		log.Fatalf("error ping database: %v", err)
	}

	log.Infof("open database with %v open connections", db.Stats().OpenConnections)
}

func closeDB() {
	log.Infof("close database with %v open connections", db.Stats().OpenConnections)

	err := db.Close()
	if err != nil {
		log.Errorf("error close database: %v", err)
	}
}

// Отримати массив ID списків відтворення та відео з БД 
func GetPlaylistWithVideo() (model.YoutubePlayLists, error) {
	log.Debugf("dbstats=%v", db.Stats())
	maxTimePublished := time.Now().Add(-*config.PeriodСollection)
	log.Debugf("get playlists with videos, maxTimePublished: %v", maxTimePublished)

	var playlists model.YoutubePlayLists = model.YoutubePlayLists{Playlists: make(map[string]*model.YoutubePlayList)}

	rows, err := db.Query(GET_PLAYLISTS_WITH_VIDEO, maxTimePublished)
	if err != nil {
		log.Errorf("Error get playlists: %v", err)
		return playlists, err
	}
	defer rows.Close()

	pl := ""
	for rows.Next() {
		var id string
		var videoId string
		var publishedat time.Time
		var title string

		rows.Scan(&id, &videoId, &publishedat, &title)
//		log.Debugf("pl: %v, video: %v, publishedat: %v, title: %v", id, videoId, publishedat, title)

		if pl != id {
			playlists.Append(id)
			pl = id
		}
		if videoId != "" {
			playlists.Playlists[id].Append(videoId, &model.YoutubeVideo{PublishedAt: publishedat,
				Title: title, Deleted: false})
		}
	}
	err = rows.Err()
	if err != nil {
		log.Error(err)
		return playlists, err
	}

	return playlists, nil
}

// Отримати массив ID списків відтворення
func GetPlaylistIDs() (map[string]bool, error) {
	log.Debugf("dbstats=%v", db.Stats())

	rows, err := db.Query(GET_PLAYLISTS)
	if err != nil {
		log.Errorf("Error get playlists: %v", err)
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
		log.Error(err)
		return nil, err
	}

	return response, nil
}

// Додати відео
func AddVideo(id, idpl string, publishedat time.Time, title, description, channelId, channelTitle string) error {
	if id == "" {
		return errors.New("Error add video, id is null")
	}
	if idpl == "" {
		return errors.New("Error add video, idpl is null")
	}

	res, err := db.Exec(INSERT_VIDEO, id, idpl, publishedat, title, description, channelId, channelTitle)
	if err != nil {
		log.Errorf("err=%v", err)
		return err
	}

	_, err = res.RowsAffected()
	if err != nil {
		log.Errorf("err=%v", err)
		return err
	} else {
		log.Debugf("insert video: id=%v, idpl=%v, publishedat=%v, title=%v, channelId=%v, channelTitle=%v", id, idpl,
			publishedat, title, channelTitle, channelId)
	}

	return nil
}

// Додати метрики
func AddMetric(metrics []*model.Metrics) error {

	txn, err := db.Begin()
	if err != nil {
		log.Errorf("err=%v", err)
		return err
	}

	stmt, err := txn.Prepare(pq.CopyIn("metric", "idvideo", "commentcount", "likecount", "dislikecount", "viewcount",
		"timemetric"))
	if err != nil {
		log.Errorf("err=%v", err)
		return err
	}

	for _, metric := range metrics {
		_, err = stmt.Exec(metric.Id, metric.CommentCount, metric.LikeCount, metric.DislikeCount, metric.ViewCount,
			metric.Time)
		if err != nil {
			log.Errorf("err=%v", err)
			return err
		}
	}

	_, err = stmt.Exec()
	if err != nil {
		log.Errorf("err=%v", err)
		return err
	}

	err = stmt.Close()
	if err != nil {
		log.Errorf("err=%v", err)
		return err
	}

	err = txn.Commit()
	if err != nil {
		log.Errorf("err=%v", err)
		return err
	}

	return nil
}

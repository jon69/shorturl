package dbh

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

func Ping(conn string) bool {
	db, err := sql.Open("postgres", conn)
	if err != nil {
		log.Println("Error connect to db: " + err.Error())
		return false
	}
	defer db.Close()
	return true
}

func CreateIfNotExist(conn string) bool {
	log.Println("CreateIfNotExist")
	db, err := sql.Open("postgres", conn)
	if err != nil {
		log.Println("Error connect to db: " + err.Error())
		return false
	}
	defer db.Close()

	// проверяем есть ли таблица в БД
	var exist bool
	queryCheck := "SELECT EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND  tablename  = 'shorturls')"
	row := db.QueryRow(queryCheck)
	err = row.Scan(&exist)
	if err != nil {
		log.Println("Error exec query [" + queryCheck + "]: " + err.Error())
		return false
	}
	// если нет, создаем
	if !exist {
		log.Println("table does not exist, creating...")
		queryCreate := "CREATE TABLE public.shorturls (uid bigserial, url bytea, originurl text unique, shorturl text, del boolean default false)"
		_, err = db.Exec(queryCreate)
		if err != nil {
			log.Println("Error exec query [" + queryCreate + "]: " + err.Error())
			return false
		}
	} else {
		log.Println("table exist")
	}
	return true
}

func InsertURL(conn string, data []byte, originURL string, shortURL string) (bool, int, string) {
	db, errOpen := sql.Open("postgres", conn)
	if errOpen != nil {
		log.Println("Error connect to db: " + errOpen.Error())
		return false, 1, ""
	}
	defer db.Close()

	insertOrUpdateQuery := `WITH e AS(
								INSERT INTO public.shorturls (url, originurl, shorturl) 
									VALUES ($1,$2,$3)
								ON CONFLICT(originurl) DO NOTHING
								RETURNING 1, uid, shorturl
							)
							SELECT * FROM e
							UNION
								SELECT 2, uid, shorturl FROM public.shorturls WHERE originurl=$2`
	var iou int
	var id int64
	var su string
	row := db.QueryRow(insertOrUpdateQuery, data, originURL, shortURL)
	err := row.Scan(&iou, &id, &su)
	if err != nil {
		log.Println("error readin from insert row: " + err.Error())
		return false, 1, su
	}

	if iou == 1 {
		log.Println("Inserted new into db: " + originURL)
	} else {
		log.Println("exist in db: " + originURL)
	}
	return true, iou, su
}

func DeleteURL(conn string, shortURL string) bool {
	db, errOpen := sql.Open("postgres", conn)
	if errOpen != nil {
		log.Println("DelUserURL | Error connect to db: " + errOpen.Error())
		return false
	}
	defer db.Close()

	queryDel := `UPDATE public.shorturls SET del=true WHERE shorturl=$1`

	_, err := db.Exec(queryDel, shortURL)
	if err != nil {
		log.Println("DelUserURL | Error exec query [" + queryDel + "]: " + err.Error())
		return false
	}
	return true
}

type URLFromDB struct {
	DumpJsonURL []byte
	Deleted     bool
}

func ReadURLS(conn string) ([]URLFromDB, bool) {
	var ret []URLFromDB
	db, errOpen := sql.Open("postgres", conn)
	if errOpen != nil {
		log.Println("ReadURLS Error connect to db: " + errOpen.Error())
		return ret, false
	}
	defer db.Close()

	rows, err := db.Query("SELECT url, del from public.shorturls")
	if err != nil {
		log.Println("Error select url: " + err.Error())
		return ret, false
	}

	// обязательно закрываем перед возвратом функции
	defer rows.Close()

	// пробегаем по всем записям
	for rows.Next() {
		var v URLFromDB
		err = rows.Scan(&v.DumpJsonURL, &v.Deleted)
		if err != nil {
			log.Println("Error rows.Scan: " + err.Error())
			return ret, false
		}
		ret = append(ret, v)
	}
	// проверяем на ошибки
	err = rows.Err()
	if err != nil {
		log.Println("Error rows.Err: " + err.Error())
		return ret, false
	}

	return ret, true
}

package main

import (
    "github.com/go-martini/martini"
    "github.com/martini-contrib/oauth2"
    "github.com/martini-contrib/sessions"
    "github.com/natefinch/sh"
    _ "github.com/mattn/go-sqlite3"
    "database/sql"

    "log"
    "net/http"
    "io/ioutil"
    "encoding/json"
    "errors"
)

func initDB() *sql.DB {
    db, err := sql.Open("sqlite3", "./go-in-the-office.sqlite")
    if err != nil {
        log.Fatal(err)
    }

    _, err = db.Exec("CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, mac_address TEXT, avatar_url TEXT);")
    if err != nil {
        log.Fatal(err)
    }

    return db
}

type User struct {
    id int
    name, avatarUrl, macAddress string
}

func (user *User) Save(db *sql.DB) error {
    _, err := db.Exec("INSERT INTO users (name, avatar_url, mac_address) VALUES(?, ?, ?)", user.name, user.avatarUrl, user.macAddress)
    if err != nil {
        return err
    }
    rows, err := db.Query("SELECT seq FROM sqlite_sequence WHERE name = ?", "users")
    defer rows.Close()
    if err != nil {
        return err
    }

    if rows.Next() {
        rows.Scan(&user.id)
    }
    return nil
}

func MacAddresses() {
    nmap := sh.Cmd("nmap")
    arp := sh.Cmd("arp")
    grep := sh.Cmd("grep")
    log.Print(nmap("10.0.1.1/24", "-sP"))
    log.Print(sh.Pipe(arp("-a"), grep("-v", "incomplete")))
}

func GetUserInfo(access_token string) (map[string]interface{}, error) {
    var result map[string]interface{}

    resp, err := http.Get("https://api.github.com/user?access_token=" + access_token)
    defer resp.Body.Close()
    if err != nil {
        return result, err
    }

    bodyBytes, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return result, err
    }
    err = json.Unmarshal(bodyBytes, &result)
    if err != nil {
        return result, err
    }

    return result, nil
}

func FindUser(db *sql.DB, userId int) (*User, error) {
    user := &User{}

    rows, err := db.Query("SELECT id, name, avatar_url, mac_address FROM users WHERE id = ?", userId)
    defer rows.Close()
    if err != nil {
        return user, err
    }

    if rows.Next() {
        rows.Scan(&user.id, &user.name, &user.avatarUrl, &user.macAddress)
    }

    return user, nil
}

func FindOrCreateUser(db *sql.DB, userInfo map[string]interface{}) (*User, error) {
    user := &User{}
    userName, okName := userInfo["login"].(string)
    userAvatar, okAvatar := userInfo["avatar_url"].(string)

    if !okName || !okAvatar {
        return user, errors.New("can not read user info")
    }

    rows, err := db.Query("SELECT id, name, avatar_url, mac_address FROM users WHERE name = ?", userName)
    defer rows.Close()
    if err != nil {
        return user, err
    }

    if rows.Next() {
        rows.Scan(&user.id, &user.name, &user.avatarUrl, &user.macAddress)
    }

    if user.id == 0 {
        user.name = userName
        user.avatarUrl = userAvatar
        user.macAddress = "60:c5:47:07:c4:bc"

        err = user.Save(db)
        if err != nil {
            return user, err
        }
    }

    return user, nil
}

//-----------------------------------------------------------------------------

func main() {
    db := initDB()
    // MacAddresses()

    m := martini.Classic()
    m.Use(sessions.Sessions("my_session", sessions.NewCookieStore([]byte("secret123"))))
    m.Use(oauth2.Github(&oauth2.Options {
        ClientId:     "187efe794fff7d76ba90",
        ClientSecret: "00a6f3bc88fcfd5ddf42217798a4495c2a99632e",
    }))

    m.Get("/", oauth2.LoginRequired, func(tokens oauth2.Tokens, session sessions.Session) string {
        userId := session.Get("userId")
        if userId != nil && !tokens.IsExpired() {
            user, err := FindUser(db, userId.(int))
            if err != nil {
                return "can not find user"
            }
            return user.avatarUrl
        } else {
            userInfo, err := GetUserInfo(tokens.Access())
            if err != nil {
                return "can not get user info"
            }

            user, err := FindOrCreateUser(db, userInfo)
            if err != nil {
                return "can not find or create user"
            }

            session.Set("userId", user.id)
            return user.avatarUrl
        }
    })

    m.Run()
}

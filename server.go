package main

import (
    "github.com/go-martini/martini"
    "github.com/martini-contrib/oauth2"
    "github.com/martini-contrib/sessions"
    "github.com/martini-contrib/render"
    "github.com/natefinch/sh"
    _ "github.com/mattn/go-sqlite3"
    "database/sql"

    "log"
    "net/http"
    "io/ioutil"
    "encoding/json"
    "errors"
    "regexp"
)

// TODO
// cache MAC addresses
// get user's MAC address

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

func RouterIp() string {
    netstat := sh.Cmd("netstat")
    grep := sh.Cmd("grep")
    netstat_output := sh.Pipe(netstat("-r", "-n"), grep("default")).String()

    re := regexp.MustCompile("([\\d.]+)")
    result := re.FindStringSubmatch(netstat_output)[1]

    return result
}

func MacAddresses() []string {
    sudo := sh.Cmd("sudo")
    grep := sh.Cmd("grep")
    nmap_output := sh.Pipe(sudo("nmap", RouterIp() + "/24", "-sP"), grep("MAC Address: ")).String()

    re := regexp.MustCompile("MAC Address: (.+) \\(.+\\)")
    mac_addresses := re.FindAllStringSubmatch(nmap_output, -1)

    result := make([]string, len(mac_addresses))
    for i, value := range mac_addresses {
        result[i] = value[1]
    }

    return result
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

func FindUserById(db *sql.DB, userId int) (*User, error) {
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

func FindUserByMacAddress(db *sql.DB, mac_address string) (*User, error) {
    user := &User{}

    rows, err := db.Query("SELECT id, name, avatar_url, mac_address FROM users WHERE mac_address = ?", mac_address)
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

    m := martini.Classic()

    m.Use(sessions.Sessions("my_session", sessions.NewCookieStore([]byte("secret123"))))
    m.Use(oauth2.Github(&oauth2.Options {
        ClientId:     "187efe794fff7d76ba90",
        ClientSecret: "00a6f3bc88fcfd5ddf42217798a4495c2a99632e",
    }))
    m.Use(render.Renderer(render.Options{
        Layout: "layout",
    }))

    m.Get("/", oauth2.LoginRequired, func(tokens oauth2.Tokens, session sessions.Session, r render.Render) {
        userId := session.Get("userId")

        if userId == nil || tokens.IsExpired() {
            userInfo, err := GetUserInfo(tokens.Access())
            if err != nil {
                r.HTML(404, "error", "can not get user info")
                return
            }

            user, err := FindOrCreateUser(db, userInfo)
            if err != nil {
                r.HTML(404, "error", "can not find or create user")
                return
            }

            session.Set("userId", user.id)
            r.HTML(200, "avatar", user.avatarUrl)

        } else {
            user, err := FindUserById(db, userId.(int))
            if err != nil {
                r.HTML(404, "error", "can not find user")
                return
            }
            r.HTML(200, "avatar", user.avatarUrl)
        }

        for _, mac_address := range MacAddresses() {
            user, err := FindUserByMacAddress(db, mac_address)
            if err == nil {
                r.HTML(200, "avatar", user.avatarUrl)
            }
        }
    })

    m.Run()
}

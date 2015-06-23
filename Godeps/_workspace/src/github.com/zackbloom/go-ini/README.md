go-ini
======

INI file decoder for Go lang.  Idea is to have an unmarshaller similar to JSON - specify parts of the file you want coded with structs and tags.

For example, for an INI file like this:

    [Pod_mysql]
    cache_size = 2000
    default_socket = /tmp/mysql.sock

    [Mysql]
    default_socket = /tmp/mysql.sock

Decode into a structure like this:

    type MyIni struct {

        PdoMysql struct {
            CacheSize     int `ini:"cache_size"`
            DefaultSocket string `ini:"default_socket"`
        } `ini:"[Pdo_myqsl]"`

        Mysql struct {
            DefaultSocket string `ini:"default_socket"`
        } `ini:"[Myqsl]"`
    }

With code like this:

    var config MyIni
    var b []byte      // config file stored here
    err := ini.Unmarshal(b, &config)


Advanced Types
==============

Over the years, INI files have grown from simple `name=value` lists of properties to files that support arrays and arrays of structures.  For example, to support playlists a music config file may look like this:

    [CREATE SONG]
    SongId=21348
    Title=Long Way to Go
    Artist=The Coach

    [CREATE SONG]
    SongId=9855
    Title=The Falcon Lead
    Artist=It Wasn't Safe

    [CREATE PLAYLIST]
    PlaylistId=438432
    Title=Acid Jazz
    Song=21348
    Song=482
    Song=9855

    [CREATE PLAYLIST]
    PlaylistId=2585
    Title=Lounge
    Song=7558
    Song=25828

With GO-INI, parsing is as simple as defining the structure and unmarshalling it.

    package main

    import (
        "encoding/json"
        "github.com/sspencer/go-ini"
        "io/ioutil"
        "log"
        "os"
    )

    type TunePlayer struct {
        Songs []struct {
            SongId int
            Title string
            Artist string
        } `ini:"[CREATE SONG]"`

        Playlists []struct {
            PlaylistId int
            Title string
            SongIds []int `ini:"Song"`
        } `ini:[CREATE PLAYLIST]`
    }

    func main() {
        var player TunePlayer

        content, err := ioutil.ReadFile("./tunes.ini")
        if err != nil {
            log.Fatal(err)
        }

        err = ini.Unmarshal(content, &player)
        if err != nil {
            log.Fatal(err)
        }

        // Output same struct as JSON to verify parsing worked
        enc := json.NewEncoder(os.Stdout)
        if err := enc.Encode(&player); err != nil {
            log.Println(err)
        }
    }





Todo
=====

Need to parse inner array of structs

    struct {
        Playlists []struct {
            Id int
            Title string
            Programs []struct {
                Id int
                Mix string
                Separation int
            } `ini:"Play Program"`
        } `ini:"[CREATE PLAYLIST]"`
    }

    [CREATE PLAYLIST]
    ID=6524
    Title=Pop
    Start Schedule

    Play Program
    ID=391
    Mix=RAND

    Play Program
    ID=3912
    Separation=10
    End Schedule

package ini

import (
	"bytes"
	"testing"
)

func TestSimple(t *testing.T) {
	var d struct {
		Start struct {
			Foo   string `ini:"FOO"`
			Magic int    `ini:"Magic Number"`
		} `ini:"[START]"`
	}

	b := []byte(`
; ignore me
[START]
FOO=BAR
Magic Number = 42`)

	err := Unmarshal(b, &d)

	if err != nil {
		t.Fatal(err)
	}

	if d.Start.Foo != "BAR" {
		t.Fatal("Field FOO not set")
	} else if d.Start.Magic != 42 {
		t.Fatal("Field Magic not set")
	}
}

func TestUnmatched(t *testing.T) {
	var d struct {
		Start struct {
			Foo   string
			Magic int
		} `ini:"[START]"`
	}

	b := []byte(`
; ignore me
[START]
FOO=BAR
UNMATCHED=ME
Magic=42`)

	ini := NewDecoder(bytes.NewReader(b))
	err := ini.Decode(&d)

	if err != nil {
		t.Fatal(err)
	}

	unmatched := ini.Unmatched()
	if d.Start.Foo != "BAR" {
		t.Fatal("FOO not set")
	} else if d.Start.Magic != 42 {
		t.Fatal("Magic not set")
	} else if len(unmatched) != 1 {
		t.Fatal("Wrong number of unmatched lines (%d): %v", len(unmatched), unmatched)
	} else if unmatched[0].line != "UNMATCHED=ME" {
		t.Fatal("Unmatched line does not match")
	}
}

func TestScalar(t *testing.T) {

	var d struct {
		Start struct {
			MyString string  `ini:"MYSTRING"`
			MyInt    int     // ini tag not required if name is same (case ignored)
			MyFloat  float64 `ini:"MYFLOAT"`
			MyBool   bool    `ini:"MYBOOL"`
		} `ini:"[START]"`
	}

	b := []byte(`
[START]
MYSTRING=hello
MYINT=234
MYFLOAT=91.4
MYBOOL=yes`)

	err := Unmarshal(b, &d)

	if err != nil {
		t.Fatal(err)
	}

	if d.Start.MyString != "hello" {
		t.Fatal("Field MyString not set")
	} else if d.Start.MyInt != 234 {
		t.Fatal("Field MyInt not set")
	} else if d.Start.MyFloat != 91.4 {
		t.Fatal("Field MyFloat not set")
	} else if d.Start.MyBool != true {
		t.Fatal("Field MyBool not set")
	}

}

func TestNoSections(t *testing.T) {

	var d struct {
		Title   string
		Version string
	}

	b := []byte(`
TITLE=Go Compiler
VERSION=1.3.3
`)

	err := Unmarshal(b, &d)

	if err != nil {
		t.Fatal(err)
	}

	if d.Title != "Go Compiler" {
		t.Fatal("Field Title not set")
	} else if d.Version != "1.3.3" {
		t.Fatal("Field Version not set")
	}
}

func TestTwoSections(t *testing.T) {

	var d struct {
		Mysql struct {
			Host string
			Port int
		} `ini:"[MYSQL]"`

		PdoMysql struct {
			Host string
		} `ini:"[PDOMYSQL]"`
	}

	b := []byte(`
[MYSQL]
HOST=mysql:localhost
[PDOMYSQL]
HOST=pdo:127.0.0.1
SOCKET=/tmp/mysql.sock
[MYSQL]
PORT=3303
`)

	err := Unmarshal(b, &d)

	if err != nil {
		t.Fatal(err)
	}

	if d.Mysql.Host != "mysql:localhost" {
		t.Fatal("Mysql Host not set")
	} else if d.Mysql.Port != 3303 {
		t.Fatal("Mysql Port not set")
	} else if d.PdoMysql.Host != "pdo:127.0.0.1" {
		t.Fatal("PDO Host not set")
	}
}

func TestDeep(t *testing.T) {

	var d struct {
		SecurePort      int    `ini:"SET OPTION SECURE AUTH PORT"`
		FallbackAddress string `ini:"SET OPTION SERVER FALLBACK ADDRESS"`
		NetworkId       int    `ini:"SET OPTION NETWORK ID"`
		Download        struct {
			MaxSpeed      int    `ini:"SET OPTION CONTENT DOWNLOAD MAX KBS"`
			DownloadStart string `ini:"SET OPTION NETWORK DOWNLOAD WINDOW START"`
			DownloadEnd   string `ini:"SET OPTION NETWORK DOWNLOAD WINDOW END"`
		} `ini:"-"`
	}
	b := []byte(`
SET OPTION SECURE AUTH PORT=8080
SET OPTION SERVER FALLBACK ADDRESS=127.0.0.1
SET OPTION CONTENT DOWNLOAD MAX KBS=56
SET OPTION NETWORK DOWNLOAD WINDOW START=22:00:00
SET OPTION NETWORK DOWNLOAD WINDOW END=23:59:00
SET OPTION NETWORK ID=53
`)

	err := Unmarshal(b, &d)

	if err != nil {
		t.Fatal(err)
	}

	if d.SecurePort != 8080 {
		t.Fatal("Secure Port does not match")
	} else if d.FallbackAddress != "127.0.0.1" {
		t.Fatal("FallbackAddress does not match")
	} else if d.NetworkId != 53 {
		t.Fatal("NetworkId does not match")
	} else if d.Download.MaxSpeed != 56 {
		t.Fatal("MaxSpeed does not match")
	} else if d.Download.DownloadStart != "22:00:00" {
		t.Fatal("DownloadStart does not match")
	} else if d.Download.DownloadEnd != "23:59:00" {
		t.Fatal("DownloadEnd does not match")
	}
}

func TestStringArray(t *testing.T) {
	var d struct {
		Playlist struct {
			Id    int
			Name  string
			Songs []string `ini:"Add Song"`
		} `ini:"[CREATE PLAYLIST]"`
	}

	b := []byte(`
[CREATE PLAYLIST]
ID=349
Name=Rock & Roll, D00d
Add Song=Time to Run
Add Song=W H O K I L L
`)

	err := Unmarshal(b, &d)

	if err != nil {
		t.Fatal(err)
	}

	if d.Playlist.Id != 349 {
		t.Fatal("Playlist Id not set")
	} else if d.Playlist.Name != "Rock & Roll, D00d" {
		t.Fatal("Playlist Name not set")
	} else if len(d.Playlist.Songs) != 2 {
		t.Fatal("Playlist Song length is incorrect")
	} else if d.Playlist.Songs[0] != "Time to Run" {
		t.Fatal("Playlist Songs[0] is incorrect")
	} else if d.Playlist.Songs[1] != "W H O K I L L" {
		t.Fatal("Playlist Songs[1] is incorrect")
	}
}

func TestIntArray(t *testing.T) {
	var d struct {
		Playlist struct {
			Id    int
			Name  string
			Songs []int `ini:"Add Song"`
		} `ini:"[CREATE PLAYLIST]"`
	}

	b := []byte(`
[CREATE PLAYLIST]
ID=349
Name=Rock & Roll, D00d
Add Song=-19
Add Song=43107
`)

	err := Unmarshal(b, &d)

	if err != nil {
		t.Fatal(err)
	}

	if d.Playlist.Id != 349 {
		t.Fatal("Playlist Id not set")
	} else if d.Playlist.Name != "Rock & Roll, D00d" {
		t.Fatal("Playlist Name not set")
	} else if len(d.Playlist.Songs) != 2 {
		t.Fatal("Playlist Song length is incorrect")
	} else if d.Playlist.Songs[0] != -19 {
		t.Fatal("Playlist Songs[0] is incorrect")
	} else if d.Playlist.Songs[1] != 43107 {
		t.Fatal("Playlist Title[1] is incorrect")
	}
}

func TestUIntArray(t *testing.T) {
	var d struct {
		Playlist struct {
			Id    uint
			Name  string
			Songs []uint `ini:"Add Song"`
		} `ini:"[CREATE PLAYLIST]"`
	}

	b := []byte(`
[CREATE PLAYLIST]
ID=349
Name=Rock & Roll, D00d
Add Song=19
Add Song=43107
`)

	err := Unmarshal(b, &d)

	if err != nil {
		t.Fatal(err)
	}

	if d.Playlist.Id != 349 {
		t.Fatal("Playlist Id not set")
	} else if d.Playlist.Name != "Rock & Roll, D00d" {
		t.Fatal("Playlist Name not set")
	} else if len(d.Playlist.Songs) != 2 {
		t.Fatal("Playlist Song length is incorrect")
	} else if d.Playlist.Songs[0] != 19 {
		t.Fatal("Playlist Songs[0] is incorrect")
	} else if d.Playlist.Songs[1] != 43107 {
		t.Fatal("Playlist Title[1] is incorrect")
	}
}

func TestFloatArray(t *testing.T) {
	var d struct {
		Playlist struct {
			Id    int
			Name  string
			Songs []float32 `ini:"Add Song"`
		} `ini:"[CREATE PLAYLIST]"`
	}

	b := []byte(`
[CREATE PLAYLIST]
ID=349
Name=Rock & Roll, D00d
Add Song=1.9
Add Song=431.7
`)

	err := Unmarshal(b, &d)

	if err != nil {
		t.Fatal(err)
	}

	if d.Playlist.Id != 349 {
		t.Fatal("Playlist Id not set")
	} else if d.Playlist.Name != "Rock & Roll, D00d" {
		t.Fatal("Playlist Name not set")
	} else if len(d.Playlist.Songs) != 2 {
		t.Fatal("Playlist Song length is incorrect")
	} else if d.Playlist.Songs[0] != 1.9 {
		t.Fatal("Playlist Songs[0] is incorrect")
	} else if d.Playlist.Songs[1] != 431.7 {
		t.Fatal("Playlist Title[1] is incorrect")
	}
}

func TestBoolArray(t *testing.T) {
	var d struct {
		Playlist struct {
			Id    int
			Name  string
			Songs []bool `ini:"Add Song"`
		} `ini:"[CREATE PLAYLIST]"`
	}

	b := []byte(`
[CREATE PLAYLIST]
ID=349
Name=Rock & Roll, D00d
Add Song=true
Add Song=false
`)

	err := Unmarshal(b, &d)

	if err != nil {
		t.Fatal(err)
	}

	if d.Playlist.Id != 349 {
		t.Fatal("Playlist Id not set")
	} else if d.Playlist.Name != "Rock & Roll, D00d" {
		t.Fatal("Playlist Name not set")
	} else if len(d.Playlist.Songs) != 2 {
		t.Fatal("Playlist Song length is incorrect")
	} else if d.Playlist.Songs[0] != true {
		t.Fatal("Playlist Songs[0] is incorrect")
	} else if d.Playlist.Songs[1] != false {
		t.Fatal("Playlist Title[1] is incorrect")
	}
}

/*
func TestArrayStruct(t *testing.T) {
	var d struct {
		Device struct {
			NumZones  int `ini:"SET OPTION ACTIVE ZONES"`
			MaxVolume int `ini:"SET OPTION ALLOW MAX VOLUME"`
		} `ini:"[ALTER DEVICE]"`
		Channels []struct {
			Title      string
			PlaylistId int `ini:"SET DEFAULT PLAYLIST"`
		} `ini:"[CREATE CHANNEL]"`
		Zones []struct {
			Volume  []float32 `ini:"SET VOLUME"`
			Channel int       `ini:"SET DEFAULT CHANNEL"`
		} `ini:"[CREATE ZONE]"`
	}

	b := []byte(`
[ALTER DEVICE]
SET OPTION ACTIVE ZONES=3
SET OPTION ALLOW MAX VOLUME=11

[CREATE CHANNEL]
TITLE=Lounge
SET DEFAULT PLAYLIST=6502

[CREATE CHANNEL]
TITLE=Acid House
SET DEFAULT PLAYLIST=4004

[CREATE ZONE]
SET VOLUME=0.65
SET DEFAULT CHANNEL=12

[CREATE ZONE]
SET VOLUME=0.18
SET VOLUME=0.55
SET DEFAULT CHANNEL=19
`)

	err := Unmarshal(b, &d)

	if err != nil {
		t.Fatal(err)
	}

	if d.Device.NumZones != 3 {
		t.Fatal("NumZones is incorrect")
	} else if d.Device.MaxVolume != 11 {
		t.Fatal("MaxVolume is incorrect")
	} else if len(d.Channels) != 2 {
		t.Fatal("Incorrect number of channels:", len(d.Channels))
	} else if d.Channels[0].Title != "Lounge" {
		t.Fatal("Channels[0] Title is incorrect")
	} else if d.Channels[0].PlaylistId != 6502 {
		t.Fatal("Channels[0] PlaylistId is incorrect")
	} else if d.Channels[1].Title != "Acid House" {
		t.Fatal("Channels[1] Title is incorrect")
	} else if d.Channels[1].PlaylistId != 4004 {
		t.Fatal("Channels[1] PlaylistId is incorrect")
	} else if len(d.Zones) != 2 {
		t.Fatal("Incorrect number of Zones:", len(d.Zones))
	} else if len(d.Zones[0].Volume) != 1 {
		t.Fatal("Zones[0] Incorrect number of Volumes:", len(d.Zones[1].Volume))
	} else if d.Zones[0].Volume[0] != 0.65 {
		t.Fatal("Zones[0] Volume is incorrect")
	} else if d.Zones[0].Channel != 12 {
		t.Fatal("Zones[0] Channel is incorrect")
	} else if len(d.Zones[1].Volume) != 2 {
		t.Fatal("Zones[1] Incorrect number of Volumes:", len(d.Zones[1].Volume))
	} else if d.Zones[1].Volume[0] != 0.18 {
		t.Fatal("Zones[1] Volume[0] is incorrect")
	} else if d.Zones[1].Volume[1] != 0.55 {
		t.Fatal("Zones[1] Volume[1] is incorrect")
	} else if d.Zones[1].Channel != 19 {
		t.Fatal("Zones[1] Channel is incorrect")
	}
}
*/

func TestStructsInStructs(t *testing.T) {
	var d struct {
		Tracks []struct {
			Id      int
			Title   string
			Sources []struct {
				Id      string
				BitRate int
			} `ini:"[CREATE AUDIO SOURCE]"`
		} `ini:"[CREATE TRACK]"`
	}

	b := []byte(`
[CREATE TRACK]
ID=82
Title=Some Song

[CREATE AUDIO SOURCE]
ID=82
BitRate=64

[CREATE AUDIO SOURCE]
ID=82
BitRate=128
`)
	err := Unmarshal(b, &d)

	if err != nil {
		t.Fatal(err)
	}

	if len(d.Tracks) != 1 {
		t.Fatal("Incorrect number of tracks,", len(d.Tracks))
	} else if len(d.Tracks[0].Sources) != 2 {
		t.Fatal("Incorrect number of sources,", len(d.Tracks[0].Sources))
	} else if d.Tracks[0].Sources[0].BitRate != 64 {
		t.Fatal("Incorrect bitrate for source[0],", d.Tracks[0].Sources[0].BitRate)
	} else if d.Tracks[0].Sources[1].BitRate != 128 {
		t.Fatal("Incorrect bitrate for source[1],", d.Tracks[0].Sources[1].BitRate)
	}
}

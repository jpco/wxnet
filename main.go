package main

import (
    "bufio"
    "fmt"
    "log"
    "net/http"
    "os"
    "strconv"
    "strings"
    "time"
)

/*
OK (request path)

uptime: (uptime from /proc/uptime)
load: (/proc/loadavg)
mem: 1328 MiB / 3871 MiB (34%) (/proc/meminfo)
 */

func fetch(path string, lines int) ([]string, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer f.Close()

    s := bufio.NewScanner(f)
    var l []string
    for len(l) < lines && s.Scan() {
        l = append(l, s.Text())
    }
    return l, s.Err()
}

func fetchUptime() (time.Duration, error) {
    uptime, err := fetch("/proc/uptime", 1)
    if err != nil {
        return 0, err
    }

    sp := strings.SplitN(uptime[0], " ", 2)
    if len(sp) == 0 {
        return 0, fmt.Errorf("oops not enough tokens")
    }

    f, err := strconv.ParseFloat(sp[0], 64)
    if err != nil {
        return 0, err
    }

    return time.Duration(f) * time.Second, nil
}

func fetchLoadavg() (string, error) {
    ls, err := fetch("/proc/loadavg", 1)
    if err != nil {
        return "", err
    }
    return ls[0], nil
}

func fetchMeminfo() (string, error) {
    meminfo, err := fetch("/proc/meminfo", 5)
    if err != nil {
        return "", err
    }
    var ks []int
    for _, l := range meminfo {
        var sp []string
        for _, t := range strings.Split(l, " ") {
            if t != "" {
                sp = append(sp, t)
            }
        }
        if len(sp) != 3 {
            return "", fmt.Errorf("oops wrong number of tokens")
        }
        k, err := strconv.Atoi(sp[1])
        if err != nil {
            return "", err
        }
        ks = append(ks, k)
    }
    used := ks[0] - ks[1] - ks[3] - ks[4]
    return fmt.Sprintf("%d MiB / %d MiB (%d%%)", used/1024, ks[0]/1024, (used*100)/ks[0]), nil
}

func putInfo(w http.ResponseWriter, r *http.Request) {
    uptime, utErr := fetchUptime()
    load, loadErr := fetchLoadavg()
    mem, memErr := fetchMeminfo()

    fmt.Fprintf(w, "<!DOCTYPE html><pre>")
    fmt.Fprintf(w, "OK %s\n\n", r.URL.Path)

    fmt.Fprintf(w, "uptime:  ")
    if utErr != nil {
        fmt.Fprintf(w, "%s\n", utErr)
    } else {
        fmt.Fprintf(w, "%s\n", uptime)
    }

    fmt.Fprintf(w, "load:    ")
    if loadErr != nil {
        fmt.Fprintf(w, "%s\n", loadErr)
    } else {
        fmt.Fprintf(w, "%s\n", load)
    }

    fmt.Fprintf(w, "mem:     ")
    if memErr != nil {
        fmt.Fprintf(w, "%s\n", memErr)
    } else {
        fmt.Fprintf(w, "%s\n", mem)
    }
}

func main() {
    http.HandleFunc("/", putInfo)

    // /etc/letsencrypt/live/badwx.jpco.io/

    go func() {
        http.ListenAndServe(":80", http.RedirectHandler("https://badwx.jpco.io/", http.StatusFound))
    }()

    log.Fatal(http.ListenAndServeTLS(":443", "/etc/letsencrypt/live/badwx.jpco.io/fullchain.pem", "/etc/letsencrypt/live/badwx.jpco.io/privkey.pem", nil))
}

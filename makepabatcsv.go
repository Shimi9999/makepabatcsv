// PABAT!の会場のURLから作品リストのCSVを生成する

package main

import (
  "os"
  "fmt"
  "strings"
  "net/http"
  "net/url"
  "encoding/csv"

  "github.com/PuerkitoBio/goquery"
)

type myError struct {
  msg string
}
func (e myError) Error() string {
  return e.msg
}

type entry struct {
  no string
  title string
  artist string
  genre string
  url string
}

func main() {
  if len(os.Args) != 2 {
    fmt.Println("Usage: makepabatcsv <pabat venue url>")
    os.Exit(1)
  }
  venueUrl := os.Args[1]

  entries, err := loadPabatEntries(venueUrl)
  if err != nil {
    panic(err)
  }

  if err := makeCsv(entries); err != nil {
    panic(err)
  }
}

func loadPabatEntries(venueUrl string) ([]entry, error) {
  vu, err := url.Parse(venueUrl)
  if err != nil {
    return nil, err
  }
  vu.Scheme = "http"

  response, err := http.Get(vu.String())
  if err != nil {
    return nil, err
  }
  defer response.Body.Close()
  if response.StatusCode != 200 {
    return nil, myError{"Status code error: " + response.Status}
  }

  doc, err := goquery.NewDocumentFromReader(response.Body)
  if err != nil {
    return nil, err
  }

  entries := make([]entry, 0)
  doc.Find(".main_body_div .pabat_readform_list_box").Each(func(index int, box *goquery.Selection) {
    var ent entry
    ent.title = box.Find(".pabat_readform_list_title_name a").Text()
    ent.url = box.Find(".pabat_readform_list_title_name a").AttrOr("href", "")
    ent.artist = box.Find(".pabat_readform_list_artist span").Text()
    ent.genre = box.Find(".pabat_readform_list_genre span").Text()

    u, err := url.Parse(ent.url)
    if err != nil {
      return
    }
    q := u.Query()
    ent.no = q.Get("num")
    u.Scheme = "http" // "https"
    u.Host = vu.Host
    u.Path = vu.Path
    ent.url = u.String()

    strs := []string{ent.title, ent.artist, ent.genre}
    for _, str := range strs {
      if strings.HasSuffix(str, "..") {
        u, err := url.Parse(ent.url)
        if err != nil {
          return
        }
        u.Scheme = "http"
        ent_, err := loadPabatEntry(u.String())
        if err != nil {
          return
        }
        ent.title = ent_.title
        ent.artist = ent_.artist
        ent.genre = ent_.genre
        break
      }
    }
    //fmt.Println(ent.no, ent.title, ent.url, ent.artist, ent.genre)
    entries = append(entries, ent)
  })

  return entries, nil
}

func loadPabatEntry(entryUrl string) (entry, error) {
  var ent entry
  response, err := http.Get(entryUrl)
  if err != nil {
    return ent, err
  }
  defer response.Body.Close()
  if response.StatusCode != 200 {
    return ent, myError{"Status code error: " + response.Status}
  }

  doc, err := goquery.NewDocumentFromReader(response.Body)
  if err != nil {
    return ent, err
  }

  ent.title = doc.Find(".pabat_readform_title").Text()
  ent.genre = doc.Find(".pabat_readform_genre").Text()
  doc.Find(".pabat_readform_border_r_2_ano .pabat_readform_w445_in_cont .pabat_readform_w445_right_text").Each(func(index int, text *goquery.Selection) {
    if index <= 1 {
      if index == 1 {
        if text.Text() == "-" {
          return
        }
        ent.artist += " / "
      }
      ent.artist += text.Text()
    }
  })

  return ent, nil
}

func makeCsv(entries []entry) error {
  records := [][]string{
    []string{"No", "GENRE", "ARTIST", "TITLE"},
  }
  for _, ent := range entries {
    title := fmt.Sprintf("=HYPERLINK(\"%s\",\"%s\")", ent.url, ent.title)
    records = append(records, []string{ent.no, ent.genre, ent.artist, title})
  }

  f, err := os.Create("./pabat.csv")
  if err != nil {
    return err
  }
  defer f.Close()

  w := csv.NewWriter(f)
  if err := w.WriteAll(records); err != nil {
    return err
  }

  return nil
}

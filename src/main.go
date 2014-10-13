package GroupMengine

import (
	"appengine"
	"appengine/urlfetch"
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	//"io/ioutil"
	"log"
	"net/http"
	//"net/url"
	"appengine/datastore"
	"appengine/taskqueue"
	//"html/template"
	"strconv"
	"strings"
)

func init() {
	http.HandleFunc("/refresh", getData)
	http.HandleFunc("/parseCar", parseCar)
	//http.HandleFunc("/", queryDb)
	//http.HandleFunc("/makes/{make:[A-Za-z]+}.json", serveMakeJson)
	http.HandleFunc("/makes.json", serveMakeJson)
	http.HandleFunc("/cars.json", getCarData)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})
}

type CarInfo struct {
	Make  string
	Model string
	Year  int
	Mpg   float64
	Url   string
}

func serveMakeJson(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	q := datastore.NewQuery("CarInfo").
		Project("Make").
		Distinct()

	var makes []string
	for t := q.Run(c); ; {
		var car2 CarInfo
		_, err := t.Next(&car2)
		if err == datastore.Done {
			break
		}
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		makes = append(makes, car2.Make)
	}

	dataJson, _ := json.Marshal(makes)

	fmt.Fprintf(w, "%s", string(dataJson))
}

func ModelScrape(client *http.Client, Make string, Model string, Url string) CarInfo {

	res, err := client.Get(Url)
	if err != nil {
		log.Fatal(err)
	}

	doc, err := goquery.NewDocumentFromResponse(res)
	if err != nil {
		log.Fatal(err)
	}

	var car CarInfo

	fmt.Printf("Parsing %s\n", Url)
	doc.Find("ul.model-year-summary").Each(func(i int, s *goquery.Selection) {

		car.Make = Make
		car.Model = Model
		s.Find(".summary-avg-data").Each(func(y int, m *goquery.Selection) {
			car.Mpg, _ = strconv.ParseFloat(m.Text(), 32)
		})
		s.Find(".summary-year").Each(func(y int, m *goquery.Selection) {
			car.Year, _ = strconv.Atoi(m.Text())
		})
		car.Url = Url
	})

	return car
}

//var tpl = template.Must(template.ParseFiles("src/templates/main.html"))

func getCarData(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	carMake := r.URL.Query()["make"]
	mpg := r.URL.Query()["mpg"]

	var orderBy string
	if len(mpg) == 1 && mpg[0] == "bottom" {
		orderBy = "Mpg"
	} else {
		orderBy = "-Mpg"
	}

	//fmt.Fprintf(w, "%+v", r.URL.Query()["make"])
	type CarDisplay struct {
		Url     string
		Display string
	}

	var dataDisplay []CarDisplay
	q := datastore.NewQuery("CarInfo")

	if len(carMake) == 1 {
		q = datastore.NewQuery("CarInfo").
			Order(orderBy).
			Filter("Make =", carMake[0]).
			Limit(10)
	} else {
		q = datastore.NewQuery("CarInfo").
			Order(orderBy).
			Limit(10)
	}

	for t := q.Run(c); ; {
		var car2 CarInfo
		_, err := t.Next(&car2)
		if err == datastore.Done {
			break
		}
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		var info CarDisplay
		info.Url = fmt.Sprintf("%s/%d", car2.Url, car2.Year)
		info.Display = fmt.Sprintf("%d %s %s (%-3.1f)", car2.Year, strings.Title(car2.Make), strings.Title(car2.Model), car2.Mpg)
		dataDisplay = append(dataDisplay, info)
	}

	dataJson, _ := json.Marshal(dataDisplay)

	fmt.Fprintf(w, "%s", string(dataJson))
}

func queryDb(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	type CarDisplay struct {
		Url     string
		Display string
	}
	type DataDisplay struct {
		Cars  []CarDisplay
		Makes []string
	}

	var dataDisplay DataDisplay

	q := datastore.NewQuery("CarInfo").
		Order("-Mpg").
		Limit(10)
	//		Project("Make").
	//		Distinct()

	for t := q.Run(c); ; {
		var car2 CarInfo
		_, err := t.Next(&car2)
		if err == datastore.Done {
			break
		}
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		var info CarDisplay
		info.Url = car2.Url
		info.Display = fmt.Sprintf("%d %s %s (%-3.1f)", car2.Year, strings.Title(car2.Make), strings.Title(car2.Model), car2.Mpg)
		dataDisplay.Cars = append(dataDisplay.Cars, info)
	}

	q = datastore.NewQuery("CarInfo").
		Project("Make").
		Distinct()

	for t := q.Run(c); ; {
		var car2 CarInfo
		_, err := t.Next(&car2)
		if err == datastore.Done {
			break
		}
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		dataDisplay.Makes = append(dataDisplay.Makes, car2.Make)
	}

	//if err := tpl.ExecuteTemplate(w, "main.html", dataDisplay); err != nil {
	//	c.Errorf("%v", err)
	//}

	fmt.Fprintf(w, "Db generated")
}

func MakeDB(r *http.Request, car chan CarInfo, countChan chan int, count int) {
	c := appengine.NewContext(r)
	for j := 0; j < count; j++ {
		jj := <-countChan
		for i := 0; i < jj; i++ {
			e := <-car
			//_, err = stmt.Exec(e.Make, e.Model, e.Year, e.Mpg)
			//fmt.Printf("Inserting %s %s %s\n", e.Make, e.Model, e.Year)
			//if err != nil {
			//	log.Fatal(err)
			//}

			_, err := datastore.Put(c, datastore.NewIncompleteKey(c, "CarInfo", nil), &e)
			if err != nil {
				return
			}
		}
	}
}

func parseCar(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	client := urlfetch.Client(c)
	url := r.FormValue("url")
	fmt.Fprintf(w, url)

	urlParts := strings.Split(url, "/")
	fmt.Fprintf(w, urlParts[0])

	var car CarInfo = ModelScrape(client, urlParts[len(urlParts)-2], urlParts[len(urlParts)-1], url)

	if len(car.Make) > 0 && len(car.Model) > 0 {
		_, err := datastore.Put(c, datastore.NewIncompleteKey(c, "CarInfo", nil), &car)
		if err != nil {
			return
		}
	}
}

func getData(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	client := urlfetch.Client(c)
	res, err := client.Get("http://www.fuelly.com/car/")
	if err != nil {
		log.Fatal(err)
	}

	doc, err := goquery.NewDocumentFromResponse(res)
	if err != nil {
		log.Fatal(err)
	}

	count := 0
	doc.Find(".models-list").Each(func(i int, s *goquery.Selection) {
		s.Find("a").Each(func(y int, m *goquery.Selection) {
			_, exists := m.Attr("href")
			if exists == true {
				count = count + 1
				//fmt.Fprintf(w, "%s\n", "%d:%s\n", count, url)
			}
		})
	})

	fmt.Fprintf(w, "Found %d\n", count)

	//car := make(chan CarInfo, 200)
	//sem := make(chan int, 20)
	//countChan := make(chan int, count)

	doc.Find(".models-list").Each(func(i int, s *goquery.Selection) {
		s.Find("a").Each(func(y int, m *goquery.Selection) {
			modelUrl, exists := m.Attr("href")
			if exists == true {

				//fmt.Fprintf(w, "%s\n", urlParts)

				t := taskqueue.NewPOSTTask("/parseCar", map[string][]string{"url": {modelUrl}})
				if _, err := taskqueue.Add(c, t, ""); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				//go ModelScrape(client, sem, car, countChan, urlParts[len(urlParts)-2], m.Text(), modelUrl)
			}
		})
	})

	fmt.Fprintf(w, "%s\n", "DB create")

	//MakeDB(r, car, countChan, count)

	fmt.Fprintf(w, "%s\n", "DB Done")
}

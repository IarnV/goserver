package main

import (
	"encoding/json"
	"net/http"
	"os"
	"sort"
	"strconv"
)

type entry struct {
	Athlete string `json:"athlete"`
	Age     int    `json:"age"`
	Country string `json:"country"`
	Year    int    `json:"year"`
	Sport   string `json:"sport"`
	Gold    int    `json:"gold"`
	Silver  int    `json:"silver"`
	Bronze  int    `json:"bronze"`
	Total   int    `json:"total"`
}

type medals struct {
	Gold   int `json:"gold"`
	Silver int `json:"silver"`
	Bronze int `json:"bronze"`
	Total  int `json:"total"`
}

type info struct {
	Athlete string            `json:"athlete"`
	Country string            `json:"country"`
	Medals  medals            `json:"medals"`
	By_year map[string]medals `json:"medals_by_year"`
}

type country struct {
	Country string `json:"country"`
	Gold    int    `json:"gold"`
	Silver  int    `json:"silver"`
	Bronze  int    `json:"bronze"`
	Total   int    `json:"total"`
}

var (
	c  map[string]string
	im map[string]map[string]info
	cm map[string]map[string]country
)

func infohand(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	m := r.URL.Query()
	name, ok := m["name"]
	if !ok {
		w.WriteHeader(404)
		w.Write([]byte(r.URL.Path))
		return
	}
	result := info{Athlete: name[0], Country: c[name[0]], By_year: make(map[string]medals)}
	for _, cur_sport := range im {
		if _, ok := cur_sport[name[0]]; ok {
			result.Medals.Gold += cur_sport[name[0]].Medals.Gold
			result.Medals.Silver += cur_sport[name[0]].Medals.Silver
			result.Medals.Bronze += cur_sport[name[0]].Medals.Bronze
			result.Medals.Total += cur_sport[name[0]].Medals.Total
			for year, year_medals := range cur_sport[name[0]].By_year {
				if _, ok := result.By_year[year]; !ok {
					result.By_year[year] = year_medals
				} else {
					new_gold, new_silver, new_bronze := result.By_year[year].Gold+year_medals.Gold, result.By_year[year].Silver+year_medals.Silver, result.By_year[year].Bronze+year_medals.Bronze
					new_medals := medals{Gold: new_gold, Silver: new_silver, Bronze: new_bronze, Total: new_gold + new_silver + new_bronze}
					result.By_year[year] = new_medals
				}
			}
		}
	}
	if len(result.By_year) == 0 {
		w.WriteHeader(404)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	json.NewEncoder(w).Encode(result)
}

func sporthand(w http.ResponseWriter, r *http.Request) {
	var limit int
	var err error
	defer r.Body.Close()
	m := r.URL.Query()
	sport_data, ok := m["sport"]
	if !ok {
		w.WriteHeader(404)
		return
	}
	sport := sport_data[0]
	limit_data, ok := m["limit"]
	if !ok {
		limit = 3
	} else {
		limit, err = strconv.Atoi(limit_data[0])
		if err != nil {
			w.WriteHeader(400)
			return
		}
	}
	if cur_sport, ok := im[sport]; ok {
		arr := make([]info, 0, len(cur_sport))
		for _, v := range cur_sport {
			arr = append(arr, v)
		}
		sort.Slice(arr, func(i, j int) bool {
			if arr[i].Medals.Gold != arr[j].Medals.Gold {
				return arr[i].Medals.Gold > arr[j].Medals.Gold
			} else if arr[i].Medals.Silver != arr[j].Medals.Silver {
				return arr[i].Medals.Silver > arr[j].Medals.Silver
			} else if arr[i].Medals.Bronze != arr[j].Medals.Bronze {
				return arr[i].Medals.Bronze > arr[j].Medals.Bronze
			} else {
				return arr[i].Athlete < arr[j].Athlete
			}
		})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(arr[:limit])
	} else {
		w.WriteHeader(404)
		return
	}
}

func countryhand(w http.ResponseWriter, r *http.Request) {
	var (
		year  string
		limit int
		err   error
	)
	defer r.Body.Close()
	m := r.URL.Query()
	year_data, ok := m["year"]
	if !ok {
		w.WriteHeader(404)
		return
	}
	year = year_data[0]
	limit_data, ok := m["limit"]
	if !ok {
		limit = 3
	} else {
		limit, err = strconv.Atoi(limit_data[0])
		if err != nil {
			w.WriteHeader(400)
			return
		}
	}
	if cur_year, ok := cm[year]; ok {
		arr := make([]country, 0, len(cur_year))
		if limit > len(cur_year) {
			limit = len(cur_year)
		}
		for _, v := range cur_year {
			arr = append(arr, v)
		}
		sort.Slice(arr, func(i, j int) bool {
			if arr[i].Gold != arr[j].Gold {
				return arr[i].Gold > arr[j].Gold
			} else if arr[i].Silver != arr[j].Silver {
				return arr[i].Silver > arr[j].Silver
			} else if arr[i].Bronze != arr[j].Bronze {
				return arr[i].Bronze > arr[j].Bronze
			} else {
				return arr[i].Country < arr[j].Country
			}
		})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(arr[:limit])
	} else {
		w.WriteHeader(404)
	}
}

func main() {
	// port := os.Args[2]
	// data_path := os.Args[4]
	port := "8080"
	data_path := "./data/sports.json"
	data, _ := os.ReadFile(data_path)
	arr := make([]entry, 0)
	err := json.Unmarshal(data, &arr)
	if err != nil {
		panic(err)
	}
	c = make(map[string]string)
	im = make(map[string]map[string]info)
	cm = make(map[string]map[string]country)
	for _, elem := range arr {
		if _, ok := c[elem.Athlete]; !ok {
			c[elem.Athlete] = elem.Country
		}
		if _, ok := im[elem.Sport]; !ok {
			im[elem.Sport] = make(map[string]info)
		}
		cur_sport := im[elem.Sport]
		if _, ok := cur_sport[elem.Athlete]; !ok {
			new_entry := info{Athlete: elem.Athlete, Country: elem.Country, Medals: medals{}, By_year: make(map[string]medals)}
			cur_sport[elem.Athlete] = new_entry
		}
		new_gold, new_silver, new_bronze := cur_sport[elem.Athlete].Medals.Gold+elem.Gold, cur_sport[elem.Athlete].Medals.Silver+elem.Silver, cur_sport[elem.Athlete].Medals.Bronze+elem.Bronze
		new_medals := medals{Gold: new_gold, Silver: new_silver, Bronze: new_bronze, Total: new_gold + new_silver + new_bronze}
		new_by_year := cur_sport[elem.Athlete].By_year
		year := strconv.Itoa(elem.Year)
		if _, ok := new_by_year[year]; !ok {
			new_by_year[year] = medals{}
		}
		new_gold, new_silver, new_bronze = new_by_year[year].Gold+elem.Gold, new_by_year[year].Silver+elem.Silver, new_by_year[year].Bronze+elem.Bronze
		new_by_year[year] = medals{Gold: new_gold, Silver: new_silver, Bronze: new_bronze, Total: new_gold + new_silver + new_bronze}
		cur_sport[elem.Athlete] = info{Athlete: elem.Athlete, Country: cur_sport[elem.Athlete].Country, Medals: new_medals, By_year: new_by_year}
		im[elem.Sport] = cur_sport

		year = strconv.Itoa(elem.Year)
		if _, ok := cm[year]; !ok {
			cm[year] = make(map[string]country)
		}
		cur_year := cm[year]
		if _, ok := cur_year[elem.Country]; !ok {
			new_entry := country{Country: elem.Country}
			cur_year[elem.Country] = new_entry
		}
		new_gold, new_silver, new_bronze = cur_year[elem.Country].Gold+elem.Gold, cur_year[elem.Country].Silver+elem.Silver, cur_year[elem.Country].Bronze+elem.Bronze
		cur_year[elem.Country] = country{Country: elem.Country, Gold: new_gold, Silver: new_silver, Bronze: new_bronze, Total: new_gold + new_silver + new_bronze}
		cm[year] = cur_year
	}
	my_serv_mux := http.NewServeMux()
	my_serv_mux.HandleFunc("/athlete-info", infohand)
	my_serv_mux.HandleFunc("/top-athletes-in-sport", sporthand)
	my_serv_mux.HandleFunc("/top-countries-in-year", countryhand)
	my_server := http.Server{Addr: "0.0.0.0:" + port, Handler: my_serv_mux}
	err = my_server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}

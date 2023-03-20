package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"
)

type restriction struct {
	Endpoint     string   `yaml:"endpoint"`
	Agents       []string `yaml:"forbidden_user_agents"`
	Forb_headers []string `yaml:"forbidden_headers"`
	Req_headers  []string `yaml:"required_headers"`
	Forb_req_re  []string `yaml:"forbidden_request_re"`
	Forb_resp_re []string `yaml:"forbidden_response_re"`
	Forb_codes   []int    `yaml:"forbidden_response_codes"`
	Req_bytes    int      `yaml:"max_request_length_bytes"`
	Resp_bytes   int      `yaml:"max_response_length_bytes"`
}

var (
	m       map[string]restriction
	servptr *string
)

func my_handler(w http.ResponseWriter, r *http.Request) {
	response_string := "Forbidden"
	c := http.Client{Transport: fire_tripper{m, http.DefaultTransport}}
	req, err := http.NewRequest(r.Method, *servptr+r.URL.RequestURI(), r.Body)
	if err != nil {
		panic(err)
	}
	for k, v := range r.Header {
		if v == nil {
			req.Header.Set(k, "")
			continue
		}
		req.Header.Set(k, v[0])
		for i := 1; i < len(v); i++ {
			req.Header.Add(k, v[i])
		}
	}
	resp, err := c.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	cur_restr := m[r.URL.Path]
	for _, code := range cur_restr.Forb_codes {
		if code == resp.StatusCode {
			w.WriteHeader(403)
			w.Write([]byte(response_string))
			return
		}
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		w.Write(nil)
		return
	}
	if cur_restr.Resp_bytes > 0 && len(body) > cur_restr.Resp_bytes {
		w.WriteHeader(403)
		w.Write([]byte(response_string))
		return
	}
	for _, bad_resp := range cur_restr.Forb_resp_re {
		if flag, err := regexp.Match(bad_resp, body); err == nil && flag {
			w.WriteHeader(403)
			w.Write([]byte(response_string))
			return
		}
	}
	for k, v := range resp.Header {
		if v == nil {
			w.Header().Set(k, "")
			continue
		}
		w.Header().Set(k, v[0])
		for i := 1; i < len(v); i++ {
			w.Header().Add(k, v[i])
		}
	}
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

func form_resp() *http.Response {
	return &http.Response{StatusCode: 403, Status: "403 Forbidden", Body: ioutil.NopCloser(bytes.NewBufferString("Forbidden"))}
}

type fire_tripper struct {
	rest_m map[string]restriction
	trip   http.RoundTripper
}

func (my_trip fire_tripper) RoundTrip(r *http.Request) (resp *http.Response, err error) {
	var (
		cur_restr restriction
		ok        bool
	)
	cur_restr, ok = my_trip.rest_m[r.URL.Path]
	if ok {
		for _, bad_agent := range cur_restr.Agents {
			if flag, err := regexp.Match(bad_agent, []byte(r.UserAgent())); err == nil && flag {
				return form_resp(), err
			}
		}
		for _, bad_header := range cur_restr.Forb_headers {
			var bad_header_name, bad_header_value string
			bad_header_slice := strings.Split(bad_header, ": ")
			if len(bad_header_slice) > 1 {
				bad_header_value = bad_header_slice[1]
			} else {
				bad_header_value = ""
			}
			bad_header_name = bad_header_slice[0]
			if bad_header_value == "" {
				if _, ok := r.Header[bad_header_name]; ok {
					return form_resp(), err
				}
			} else {
				for _, cur_header_value := range r.Header[bad_header_name] {
					if bad_header_value == cur_header_value {
						return form_resp(), err
					}
				}
			}
		}
		for _, good_header := range cur_restr.Req_headers {
			var good_header_name, good_header_value string
			good_header_slice := strings.Split(good_header, ": ")
			if len(good_header_slice) > 1 {
				good_header_value = good_header_slice[1]
			} else {
				good_header_value = ""
			}
			good_header_name = good_header_slice[0]
			if good_header_value == "" {
				if _, ok := r.Header[good_header_name]; !ok {
					return form_resp(), err
				}
			} else {
				if _, ok := r.Header[good_header_name]; !ok {
					return form_resp(), err
				}
				check := false
				for _, cur_header_value := range r.Header[good_header_name] {
					if good_header_value == cur_header_value {
						check = true
						break
					}
				}
				if !check {
					return form_resp(), err
				}
			}
		}
		if r.Body != nil {
			body_val, err := ioutil.ReadAll(r.Body)
			r.Body.Close()
			r.Body = ioutil.NopCloser(bytes.NewBuffer(body_val))
			if err == nil {
				for _, bad_req := range cur_restr.Forb_req_re {
					if flag, err := regexp.Match(bad_req, body_val); err == nil && flag {
						return form_resp(), err
					}
				}
			}
			if len(body_val) > cur_restr.Req_bytes && cur_restr.Req_bytes > 0 {
				return form_resp(), nil
			}
		}
	}
	return my_trip.trip.RoundTrip(r)
}

func main() {
	var mm map[string][]restriction
	m = make(map[string]restriction)
	servptr = flag.String("service-addr", "http://sports:8080", "falf")
	confptr := flag.String("conf", "./example.yaml", "fd")
	addrptr := flag.String("addr", "0.0.0.0:8081", "saf")
	flag.Parse()
	data, err := os.ReadFile(*confptr)
	if err != nil {
		log.Fatalf("cannot read file: %v", err)
	}
	err = yaml.UnmarshalStrict(data, &mm)
	if err != nil {
		log.Fatalf("cannot unmarshal data: %v", err)
	}
	for _, cur_struct := range mm["rules"] {
		m[cur_struct.Endpoint] = cur_struct
	}
	my_servmux := http.NewServeMux()
	my_servmux.HandleFunc("/", my_handler)
	http.ListenAndServe(*addrptr, my_servmux)
}

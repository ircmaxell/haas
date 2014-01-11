package main

import (
    "io"
    "bytes"
    "fmt"
    "strings"
    "net/http"
    "text/template"
    "encoding/json"
)

type HugType int
type ResponseType int

const (
    hug_Hug HugType = iota
    hug_BearHug
    hug_GroupHug
)

const (
    response_HTML ResponseType = iota
    response_Text
    response_JSON
)

type HugRequest struct {
    To, From string
    Hug HugType
}

type jsonResponse struct {
    Message string
}

func init() {
    http.HandleFunc("/hug/", handleHug)
    http.HandleFunc("/bearhug/", handleBearHug)
    http.HandleFunc("/grouphug/", handleGroupHug)
}

func handleHug(w http.ResponseWriter, r *http.Request) {
    names := extractNames("/hug/", r)
    if len(names) < 2 {
        send400(w)
        return
    }
    hug := HugRequest{names[0], names[1], hug_Hug}
    sendResponse(w, r, hug)
}

func handleGroupHug(w http.ResponseWriter, r *http.Request) {
    names := extractNames("/grouphug/", r)
    if len(names) < 2 {
        send400(w)
        return
    }
    to := parseCommaSeparatedString(names[0])
    from := parseCommaSeparatedString(names[1])
    t := hug_Hug
    if strings.Contains(names[1], ",") {
        t = hug_GroupHug
    }
    hug := HugRequest{to, from, t}
    sendResponse(w, r, hug)
}

func parseCommaSeparatedString(in string) string {
    if !strings.Contains(in, ",") {
        return in
    }
    parts := strings.Split(in, ",")
    list := strings.Join(parts[0:len(parts)-1], ", ")
    return fmt.Sprintf("%s and %s", list, parts[len(parts)-1])
}

func handleBearHug(w http.ResponseWriter, r *http.Request) {
    names := extractNames("/bearhug/", r)
    if len(names) < 2 {
        send400(w)
        return
    }
    hug := HugRequest{names[0], names[1], hug_BearHug}
    sendResponse(w, r, hug)
}

func extractNames(stub string, r *http.Request) []string {
    path := r.URL.Path[len(stub):]
    names := strings.Split(path, "/")
    return names
}

func send400(w http.ResponseWriter) {
    w.Header().Set("Status", "400")
    fmt.Fprintf(w, "400 Bad Request")
}

func sendResponse(w http.ResponseWriter, r *http.Request, hug HugRequest) {
    resp := determineResponseType(r)
    if (resp == response_JSON) {
        sendResponseJSON(w, hug)
        return
    }
    temp := getTemplateName(hug.Hug, resp)
    executeTemplate(w, temp, hug)
    switch resp {
        case response_HTML:
            w.Header().Set("Content-Type", "text/html")
        case response_Text:
            w.Header().Set("Content-Type", "text/plain")
        case response_JSON:
            
    }
}

func sendResponseJSON(w http.ResponseWriter, hug HugRequest) {
    var writeBuffer bytes.Buffer
    var resp jsonResponse
    w.Header().Set("Content-Type", "application/json")

    temp := getTemplateName(hug.Hug, response_Text)
    executeTemplate(&writeBuffer, temp, hug)
    resp.Message = writeBuffer.String()
    encoder := json.NewEncoder(w)
    encoder.Encode(&resp)
}

func executeTemplate(w io.Writer, templateName string, hug HugRequest) {
    tmpl, err := template.ParseFiles(templateName)
    if err != nil {
        panic(err)
    }
    err = tmpl.Execute(w, hug)
    if err != nil {
        panic(err)
    }
}

func determineResponseType(r *http.Request) ResponseType {
    accept := r.Header.Get("Accept")
    parts := strings.Split(accept, ",")
    for _, t := range parts {
        switch {
            case strings.Contains(t, "text/html"):
                return response_HTML
            case strings.Contains(t, "text/plain"):
                return response_Text
            case strings.Contains(t, "application/json"):
                return response_JSON
        } 
    }
    return response_HTML
}

func getTemplateName(hug HugType, resp ResponseType) string {
    var resp_type, hug_type string = "", ""
    switch resp {
        case response_Text:
            resp_type = "text"
        case response_HTML:
            resp_type = "html"
    }
    switch hug {
        case hug_Hug:
            hug_type = "hug"
        case hug_BearHug:
            hug_type = "bearhug"
        case hug_GroupHug:
            hug_type = "grouphug"
    }
    return fmt.Sprintf("templates/%s_%s", resp_type, hug_type)  
}
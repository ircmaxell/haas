package main

import (
    "io"
    "bytes"
    "fmt"
    "strings"
    "net/http"
    textTemplate "text/template"
    htmlTemplate "html/template"
    "encoding/json"
    "path/filepath"
    "io/ioutil"
    "gopkg.in/yaml.v2"
)

type Languages struct {
    Languages map[string] string
}

type Formatter struct {
    Name string
    ContentType string
    GetTemplate func(hug HugRequest) TemplateExecute
    Render func(TemplateExecute, HugRequest, io.Writer)
}

type HugHandler struct {
    Path string
    Handler func(HugRequest, http.ResponseWriter)
    Template string
    MinNames int
}

type HugRequest struct {
    To, From string
    HugArray []string
    Request *http.Request
    Handler HugHandler
    Template string
    Config Configuration
}

type Configuration struct {
    Handlers map[string]HugHandler
    Formatters map[string]Formatter
}

type TemplateExecute interface {
    Execute(io.Writer, interface{}) error
}

func init() {
    config := Configuration{declareHandlers(), declareFormatters()}
    for _, handler := range config.Handlers {
        http.HandleFunc(handler.Path, getHandler(handler, config))
    }
}

func declareFormatters() map[string]Formatter {
    formatters := map[string]Formatter{
        "html": {"html", "text/html", getHtmlTemplate, renderDefault},
        "text": {"text", "text/plain", getTextTemplate, renderDefault},
        "json": {"json", "application/json", getTextTemplate, func(tmpl TemplateExecute, hug HugRequest, w io.Writer) {
            var writeBuffer bytes.Buffer
            renderDefault(tmpl, hug, &writeBuffer)
            encoder := json.NewEncoder(w)
            encoder.Encode(map[string]string {
                "message": writeBuffer.String(),
            })
        }},
    }
    return formatters
}

func declareHandlers() map[string]HugHandler {
    handlers := map[string]HugHandler{
        "hug": {
            "/hug/",
            handleGenericHug,
            "hug",
            2,
        },
        "bearhug": {
            "/bearhug/",
            handleGenericHug,
            "bearhug",
            2,
        },
        "hugattack": {
            "/hugattack/",
            handleGenericHug,
            "hugattack",
            1,
        },
        "grouphug": {
            "/grouphug/",
            func(hug HugRequest, w http.ResponseWriter) {
                hug.From = parseCommaSeparatedString(hug.From)
                hug.To = parseCommaSeparatedString(hug.To)
                if strings.Contains(hug.From, ",") {
                    hug.Template = "hug"
                }
                handleGenericHug(hug, w)
            },
            "grouphug",
            2,
        },
    }
    return handlers
}

func getHandler(h HugHandler, config Configuration) func(http.ResponseWriter, *http.Request) {
    return func(w http.ResponseWriter, r *http.Request) {
        path := r.URL.Path[len(h.Path):]
        names := strings.Split(path, "/")
        if len(names) < h.MinNames {
            w.Header().Set("Status", "400")
            fmt.Fprintf(w, "400 Bad Request")
            return
        }
        var from, to string = "", ""
        if len(names) >= 1 {
            to = names[0]
        }
        if len(names) >= 2 {
            from = names[1]
        }

        hug := HugRequest{to, from, getHugArray(r),r, h, h.Template, config}
        h.Handler(hug, w)
    }
}

func handleGenericHug(hug HugRequest, w http.ResponseWriter) {
    formatter := findFormatter(hug)
    w.Header().Set("Content-Type", formatter.Name)
    formatter.Render(formatter.GetTemplate(hug), hug, w)
}

func parseCommaSeparatedString(in string) string {
    if !strings.Contains(in, ",") {
        return in
    }
    parts := strings.Split(in, ",")
    list := strings.Join(parts[0:len(parts)-1], ", ")
    return fmt.Sprintf("%s and %s", list, parts[len(parts)-1])
}

func getHeaderOverride(header string, r *http.Request) string {
    value := r.Header.Get(header)
    r.ParseForm()
    tmp := r.Form.Get(header)
    if tmp != "" {
        return tmp
    }
    return value
}

func findFormatter(hug HugRequest) Formatter {
    accept := getHeaderOverride("Accept", hug.Request)
    parts := strings.Split(accept, ",")
    for _, t := range parts {
        for _, h := range hug.Config.Formatters {
            if strings.Contains(t, h.ContentType) {
                return h
            }
        }
    }
    return hug.Config.Formatters["html"]
}

func handleTemplateError(err error) {
    if err != nil {
        // TODO: Actually handle the error!!! :-D
        panic(err)
    }
}

func renderDefault(tmpl TemplateExecute, hug HugRequest, w io.Writer) {
    err := tmpl.Execute(w, hug)
    handleTemplateError(err)
}

func getTextTemplate(hug HugRequest) TemplateExecute {
    tmpl, err := textTemplate.ParseFiles(fmt.Sprintf("templates/%s.text", hug.Template))
    handleTemplateError(err)
    return tmpl
}

func getHtmlTemplate(hug HugRequest) TemplateExecute {
    tmpl, err := htmlTemplate.ParseFiles(fmt.Sprintf("templates/%s.html", hug.Template))
    handleTemplateError(err)
    return tmpl
}

func getHugArray(r *http.Request) []string {
    language := r.URL.Query().Get("language")
    filename, _ := filepath.Abs("./languages.yml")
    yamlFile, err := ioutil.ReadFile(filename)

    if err != nil {
        panic(err)
    }

    var lang Languages

    err = yaml.Unmarshal(yamlFile, &lang)

    if err != nil {
        panic(err)
    }

    ha := []string{}

    hs, ok := lang.Languages[language]

    if ok {
        ha = append(ha, hs)
    }

    if "" == language {
        for _, v := range lang.Languages {
            ha = append(ha, v)
        }
    }

    return ha
}

package handler

import (
	"net/http"
	"os"
)

func SwaggerYAML(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile("swagger.yaml")
	if err != nil {
		http.Error(w, "swagger.yaml পাওয়া যায়নি", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/yaml")
	w.Write(data)
}

func SwaggerUI(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head>
  <title>Chai Shop API</title>
  <link rel="icon" href="data:image/svg+xml,<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 100 100'><text y='.9em' font-size='90'>☕</text></svg>">
  <meta charset="utf-8"/>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist/swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({
      url: "/swagger.yaml",
      dom_id: '#swagger-ui',
      presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
      layout: "BaseLayout"
    })
  </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}
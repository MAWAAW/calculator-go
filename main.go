package main

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"

	_ "modernc.org/sqlite"
)

var (
	db  *sql.DB
	tpl *template.Template
)

func main() {
	initDB()

	// Charge le template HTML
	tpl = template.Must(template.ParseFiles("templates/index.html"))

	// Sert les fichiers statiques (CSS, JS)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Page principale
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tpl.Execute(w, nil)
	})

	// API sécurisée pour toutes les opérations
	http.HandleFunc("/api/calc", secureHandler(calcHandler))

	log.Println("Server Running at http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}

// Middleware de sécurité minimal
func secureHandler(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Headers de sécurité
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'self'")

		// Limite la taille de la requête (anti DoS)
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 Mo max

		next(w, r)
	}
}

// Handler pour effectuer les opérations
func calcHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	type requestData struct {
		A  float64 `json:"a"`
		B  float64 `json:"b"`
		Op string  `json:"op"`
	}

	var data requestData
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, "Entrée invalide", http.StatusBadRequest)
		return
	}

	var result float64
	switch data.Op {
	case "add":
		result = data.A + data.B
	case "sub":
		result = data.A - data.B
	case "mul":
		result = data.A * data.B
	case "div":
		if data.B == 0 {
			http.Error(w, "Division par zéro impossible", http.StatusBadRequest)
			return
		}
		result = data.A / data.B
	default:
		http.Error(w, "Opération inconnue", http.StatusBadRequest)
		return
	}

	// Stockage dans SQLite
	if err := insertOperation(data.Op, data.A, data.B, result); err != nil {
		http.Error(w, "Erreur stockage DB", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]float64{"result": result})
}

// -------------------
// SQLite + init DB
// -------------------
func initDB() {
	// Crée le dossier "data" si nécessaire
	if _, err := os.Stat("data"); os.IsNotExist(err) {
		err := os.Mkdir("data", 0755)
		if err != nil {
			log.Fatal("Impossible de créer le dossier data:", err)
		}
	}

	// Ouvre ou crée la DB
	var err error
	db, err = sql.Open("sqlite", "./data/app.db")
	if err != nil {
		log.Fatal(err)
	}

	db.SetMaxOpenConns(1)

	// Crée la table si elle n'existe pas
	query := `CREATE TABLE IF NOT EXISTS history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		op TEXT NOT NULL,
		a REAL NOT NULL,
		b REAL NOT NULL,
		result REAL NOT NULL
	);`

	_, err = db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
}

// Ajoute une opération dans l’historique
func insertOperation(op string, a, b, res float64) error {
	stmt, err := db.Prepare("INSERT INTO history(op, a, b, result) VALUES(?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(op, a, b, res)
	return err
}

package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"strconv"

	_ "github.com/mattn/go-sqlite3"
)

type Partner struct {
	ID               int
	OrganizationName string
	Address          string
	ContactPerson    string
	Phone            string
	Email            string
	PartnershipType  string
}

type PageData struct {
	Query    string
	Partners []Partner
}

var db *sql.DB

func main() {
	var err error
	db, err = sql.Open("sqlite3", "./database.db")
	if err != nil {
		log.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}
	if err := createTable(); err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", listPartners)
	http.HandleFunc("/add", addPartner)
	http.HandleFunc("/edit", editPartner)
	http.HandleFunc("/delete", deletePartner)

	log.Println("Сервер: http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func createTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS social_partners (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		organization_name TEXT NOT NULL,
		address TEXT,
		contact_person TEXT,
		phone TEXT,
		email TEXT,
		partnership_type TEXT NOT NULL
	);`
	_, err := db.Exec(query)
	return err
}

func listPartners(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")

	var rows *sql.Rows
	var err error

	if q == "" {
		rows, err = db.Query(`SELECT id, organization_name, address, contact_person, phone, email, partnership_type
			FROM social_partners ORDER BY id DESC`)
	} else {
		rows, err = db.Query(`SELECT id, organization_name, address, contact_person, phone, email, partnership_type
			FROM social_partners
			WHERE organization_name LIKE ? OR partnership_type LIKE ? OR contact_person LIKE ?
			ORDER BY id DESC`, "%"+q+"%", "%"+q+"%", "%"+q+"%")
	}

	if err != nil {
		http.Error(w, "DB error: "+err.Error(), 500)
		return
	}
	defer rows.Close()

	var partners []Partner
	for rows.Next() {
		var p Partner
		if err := rows.Scan(&p.ID, &p.OrganizationName, &p.Address, &p.ContactPerson, &p.Phone, &p.Email, &p.PartnershipType); err != nil {
			http.Error(w, "Scan error: "+err.Error(), 500)
			return
		}
		partners = append(partners, p)
	}

	tmpl, err := template.ParseFiles("templates/partners.html")
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), 500)
		return
	}
	tmpl.Execute(w, PageData{Query: q, Partners: partners})
}

func addPartner(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		p := Partner{
			OrganizationName: r.FormValue("organization_name"),
			PartnershipType:  r.FormValue("partnership_type"),
			Address:          r.FormValue("address"),
			ContactPerson:    r.FormValue("contact_person"),
			Phone:            r.FormValue("phone"),
			Email:            r.FormValue("email"),
		}

		if p.OrganizationName == "" || p.PartnershipType == "" {
			http.Error(w, "Заполните обязательные поля", 400)
			return
		}

		_, err := db.Exec(`INSERT INTO social_partners
			(organization_name, address, contact_person, phone, email, partnership_type)
			VALUES (?, ?, ?, ?, ?, ?)`,
			p.OrganizationName, p.Address, p.ContactPerson, p.Phone, p.Email, p.PartnershipType,
		)
		if err != nil {
			http.Error(w, "Insert error: "+err.Error(), 500)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	tmpl, err := template.ParseFiles("templates/add_partner.html")
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), 500)
		return
	}
	tmpl.Execute(w, nil)
}

func editPartner(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "Неверный id", 400)
		return
	}

	if r.Method == http.MethodPost {
		p := Partner{
			ID:               id,
			OrganizationName: r.FormValue("organization_name"),
			PartnershipType:  r.FormValue("partnership_type"),
			Address:          r.FormValue("address"),
			ContactPerson:    r.FormValue("contact_person"),
			Phone:            r.FormValue("phone"),
			Email:            r.FormValue("email"),
		}

		if p.OrganizationName == "" || p.PartnershipType == "" {
			http.Error(w, "Заполните обязательные поля", 400)
			return
		}

		_, err := db.Exec(`UPDATE social_partners SET
			organization_name=?,
			address=?,
			contact_person=?,
			phone=?,
			email=?,
			partnership_type=?
			WHERE id=?`,
			p.OrganizationName, p.Address, p.ContactPerson, p.Phone, p.Email, p.PartnershipType, p.ID,
		)
		if err != nil {
			http.Error(w, "Update error: "+err.Error(), 500)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// GET: загрузить данные партнёра
	var p Partner
	err = db.QueryRow(`SELECT id, organization_name, address, contact_person, phone, email, partnership_type
		FROM social_partners WHERE id=?`, id).
		Scan(&p.ID, &p.OrganizationName, &p.Address, &p.ContactPerson, &p.Phone, &p.Email, &p.PartnershipType)

	if err == sql.ErrNoRows {
		http.Error(w, "Партнёр не найден", 404)
		return
	}
	if err != nil {
		http.Error(w, "DB error: "+err.Error(), 500)
		return
	}

	tmpl, err := template.ParseFiles("templates/edit_partner.html")
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), 500)
		return
	}
	tmpl.Execute(w, p)
}

func deletePartner(w http.ResponseWriter, r *http.Request) {
	// удаление только POST, чтобы случайно не удалить по ссылке
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", 405)
		return
	}

	idStr := r.FormValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "Неверный id", 400)
		return
	}

	_, err = db.Exec("DELETE FROM social_partners WHERE id=?", id)
	if err != nil {
		http.Error(w, "Delete error: "+err.Error(), 500)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

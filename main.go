package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"strconv"

	_ "github.com/mattn/go-sqlite3"
)

// ─── MODELS ──────────────────────────────────────────────────────────────────

type Partner struct {
	ID               int
	OrganizationName string
	Address          string
	ContactPerson    string
	Phone            string
	Email            string
	PartnershipType  string
	MemoDate         string
}

type Event struct {
	ID          int
	Title       string
	Location    string
	EventDate   string
	EventTime   string
	Category    string
	Description string
	CreatedAt   string
}

type Job struct {
	ID           int
	Title        string
	Company      string
	JobType      string // Жұмыс / Тағылымдама
	Requirements string
	Description  string
	Deadline     string
	CreatedAt    string
}

// ─── PAGE DATA ───────────────────────────────────────────────────────────────

type PageData struct {
	Query    string
	Partners []Partner
}

type EventPageData struct {
	Query  string
	Events []Event
}

type JobPageData struct {
	Query string
	Jobs  []Job
}

var db *sql.DB

// ─── MAIN ────────────────────────────────────────────────────────────────────

func main() {
	var err error
	db, err = sql.Open("sqlite3", "./database.db")
	if err != nil {
		log.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}
	if err := createTables(); err != nil {
		log.Fatal(err)
	}

	// Partners
	http.HandleFunc("/", listPartners)
	http.HandleFunc("/add", addPartner)
	http.HandleFunc("/edit", editPartner)
	http.HandleFunc("/delete", deletePartner)

	// Events
	http.HandleFunc("/events", listEvents)
	http.HandleFunc("/events/add", addEvent)
	http.HandleFunc("/events/edit", editEvent)
	http.HandleFunc("/events/delete", deleteEvent)

	// Jobs
	http.HandleFunc("/jobs", listJobs)
	http.HandleFunc("/jobs/add", addJob)
	http.HandleFunc("/jobs/edit", editJob)
	http.HandleFunc("/jobs/delete", deleteJob)

	log.Println("Сервер: http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// ─── TABLES ──────────────────────────────────────────────────────────────────

func createTables() error {
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS social_partners (
		id                INTEGER PRIMARY KEY AUTOINCREMENT,
		organization_name TEXT NOT NULL,
		address           TEXT,
		contact_person    TEXT,
		phone             TEXT,
		email             TEXT,
		partnership_type  TEXT NOT NULL,
		memo_date         TEXT
	);`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS events (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		title       TEXT NOT NULL,
		location    TEXT,
		event_date  TEXT,
		event_time  TEXT,
		category    TEXT,
		description TEXT,
		created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
	);`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS jobs (
		id           INTEGER PRIMARY KEY AUTOINCREMENT,
		title        TEXT NOT NULL,
		company      TEXT,
		job_type     TEXT,
		requirements TEXT,
		description  TEXT,
		deadline     TEXT,
		created_at   DATETIME DEFAULT CURRENT_TIMESTAMP
	);`)
	return err
}

// ─── PARTNERS ────────────────────────────────────────────────────────────────

func listPartners(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	var rows *sql.Rows
	var err error

	if q == "" {
		rows, err = db.Query(`SELECT id, organization_name, address, contact_person, phone, email, partnership_type, COALESCE(memo_date,'')
			FROM social_partners ORDER BY id ASC`)
	} else {
		rows, err = db.Query(`SELECT id, organization_name, address, contact_person, phone, email, partnership_type, COALESCE(memo_date,'')
			FROM social_partners
			WHERE organization_name LIKE ? OR partnership_type LIKE ? OR contact_person LIKE ?
			ORDER BY id ASC`, "%"+q+"%", "%"+q+"%", "%"+q+"%")
	}
	if err != nil {
		http.Error(w, "DB error: "+err.Error(), 500)
		return
	}
	defer rows.Close()

	var partners []Partner
	for rows.Next() {
		var p Partner
		if err := rows.Scan(&p.ID, &p.OrganizationName, &p.Address, &p.ContactPerson,
			&p.Phone, &p.Email, &p.PartnershipType, &p.MemoDate); err != nil {
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
			MemoDate:         r.FormValue("memo_date"),
		}
		if p.OrganizationName == "" {
			http.Error(w, "Заполните обязательные поля", 400)
			return
		}
		_, err := db.Exec(`INSERT INTO social_partners
			(organization_name, address, contact_person, phone, email, partnership_type, memo_date)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			p.OrganizationName, p.Address, p.ContactPerson, p.Phone, p.Email, p.PartnershipType, p.MemoDate,
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
			MemoDate:         r.FormValue("memo_date"),
		}
		if p.OrganizationName == "" {
			http.Error(w, "Заполните обязательные поля", 400)
			return
		}
		_, err := db.Exec(`UPDATE social_partners SET
			organization_name=?, address=?, contact_person=?, phone=?, email=?, partnership_type=?, memo_date=?
			WHERE id=?`,
			p.OrganizationName, p.Address, p.ContactPerson, p.Phone, p.Email, p.PartnershipType, p.MemoDate, p.ID,
		)
		if err != nil {
			http.Error(w, "Update error: "+err.Error(), 500)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	var p Partner
	err = db.QueryRow(`SELECT id, organization_name, address, contact_person, phone, email, partnership_type, COALESCE(memo_date,'')
		FROM social_partners WHERE id=?`, id).
		Scan(&p.ID, &p.OrganizationName, &p.Address, &p.ContactPerson, &p.Phone, &p.Email, &p.PartnershipType, &p.MemoDate)
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

// ─── EVENTS ──────────────────────────────────────────────────────────────────

func listEvents(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	var rows *sql.Rows
	var err error

	if q == "" {
		rows, err = db.Query(`SELECT id, title, COALESCE(location,''), COALESCE(event_date,''), COALESCE(event_time,''), COALESCE(category,''), COALESCE(description,''), created_at
			FROM events ORDER BY event_date ASC`)
	} else {
		rows, err = db.Query(`SELECT id, title, COALESCE(location,''), COALESCE(event_date,''), COALESCE(event_time,''), COALESCE(category,''), COALESCE(description,''), created_at
			FROM events WHERE title LIKE ? OR category LIKE ? OR location LIKE ?
			ORDER BY event_date ASC`, "%"+q+"%", "%"+q+"%", "%"+q+"%")
	}
	if err != nil {
		http.Error(w, "DB error: "+err.Error(), 500)
		return
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.Title, &e.Location, &e.EventDate, &e.EventTime,
			&e.Category, &e.Description, &e.CreatedAt); err != nil {
			http.Error(w, "Scan error: "+err.Error(), 500)
			return
		}
		events = append(events, e)
	}

	tmpl, err := template.ParseFiles("templates/events.html")
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), 500)
		return
	}
	tmpl.Execute(w, EventPageData{Query: q, Events: events})
}

func addEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		e := Event{
			Title:       r.FormValue("title"),
			Location:    r.FormValue("location"),
			EventDate:   r.FormValue("event_date"),
			EventTime:   r.FormValue("event_time"),
			Category:    r.FormValue("category"),
			Description: r.FormValue("description"),
		}
		if e.Title == "" {
			http.Error(w, "Заполните обязательные поля", 400)
			return
		}
		_, err := db.Exec(`INSERT INTO events (title, location, event_date, event_time, category, description)
			VALUES (?, ?, ?, ?, ?, ?)`,
			e.Title, e.Location, e.EventDate, e.EventTime, e.Category, e.Description,
		)
		if err != nil {
			http.Error(w, "Insert error: "+err.Error(), 500)
			return
		}
		http.Redirect(w, r, "/events", http.StatusSeeOther)
		return
	}
	tmpl, err := template.ParseFiles("templates/add_event.html")
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), 500)
		return
	}
	tmpl.Execute(w, nil)
}

func editEvent(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "Неверный id", 400)
		return
	}

	if r.Method == http.MethodPost {
		e := Event{
			ID:          id,
			Title:       r.FormValue("title"),
			Location:    r.FormValue("location"),
			EventDate:   r.FormValue("event_date"),
			EventTime:   r.FormValue("event_time"),
			Category:    r.FormValue("category"),
			Description: r.FormValue("description"),
		}
		if e.Title == "" {
			http.Error(w, "Заполните обязательные поля", 400)
			return
		}
		_, err := db.Exec(`UPDATE events SET title=?, location=?, event_date=?, event_time=?, category=?, description=?
			WHERE id=?`,
			e.Title, e.Location, e.EventDate, e.EventTime, e.Category, e.Description, e.ID,
		)
		if err != nil {
			http.Error(w, "Update error: "+err.Error(), 500)
			return
		}
		http.Redirect(w, r, "/events", http.StatusSeeOther)
		return
	}

	var e Event
	err = db.QueryRow(`SELECT id, title, COALESCE(location,''), COALESCE(event_date,''), COALESCE(event_time,''), COALESCE(category,''), COALESCE(description,''), created_at
		FROM events WHERE id=?`, id).
		Scan(&e.ID, &e.Title, &e.Location, &e.EventDate, &e.EventTime, &e.Category, &e.Description, &e.CreatedAt)
	if err == sql.ErrNoRows {
		http.Error(w, "Событие не найдено", 404)
		return
	}
	if err != nil {
		http.Error(w, "DB error: "+err.Error(), 500)
		return
	}

	tmpl, err := template.ParseFiles("templates/edit_event.html")
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), 500)
		return
	}
	tmpl.Execute(w, e)
}

func deleteEvent(w http.ResponseWriter, r *http.Request) {
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
	_, err = db.Exec("DELETE FROM events WHERE id=?", id)
	if err != nil {
		http.Error(w, "Delete error: "+err.Error(), 500)
		return
	}
	http.Redirect(w, r, "/events", http.StatusSeeOther)
}

// ─── JOBS ────────────────────────────────────────────────────────────────────

func listJobs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	var rows *sql.Rows
	var err error

	if q == "" {
		rows, err = db.Query(`SELECT id, title, COALESCE(company,''), COALESCE(job_type,''), COALESCE(requirements,''), COALESCE(description,''), COALESCE(deadline,''), created_at
			FROM jobs ORDER BY id DESC`)
	} else {
		rows, err = db.Query(`SELECT id, title, COALESCE(company,''), COALESCE(job_type,''), COALESCE(requirements,''), COALESCE(description,''), COALESCE(deadline,''), created_at
			FROM jobs WHERE title LIKE ? OR company LIKE ? OR job_type LIKE ?
			ORDER BY id DESC`, "%"+q+"%", "%"+q+"%", "%"+q+"%")
	}
	if err != nil {
		http.Error(w, "DB error: "+err.Error(), 500)
		return
	}
	defer rows.Close()

	var jobs []Job
	for rows.Next() {
		var j Job
		if err := rows.Scan(&j.ID, &j.Title, &j.Company, &j.JobType,
			&j.Requirements, &j.Description, &j.Deadline, &j.CreatedAt); err != nil {
			http.Error(w, "Scan error: "+err.Error(), 500)
			return
		}
		jobs = append(jobs, j)
	}

	tmpl, err := template.ParseFiles("templates/jobs.html")
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), 500)
		return
	}
	tmpl.Execute(w, JobPageData{Query: q, Jobs: jobs})
}

func addJob(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		j := Job{
			Title:        r.FormValue("title"),
			Company:      r.FormValue("company"),
			JobType:      r.FormValue("job_type"),
			Requirements: r.FormValue("requirements"),
			Description:  r.FormValue("description"),
			Deadline:     r.FormValue("deadline"),
		}
		if j.Title == "" {
			http.Error(w, "Заполните обязательные поля", 400)
			return
		}
		_, err := db.Exec(`INSERT INTO jobs (title, company, job_type, requirements, description, deadline)
			VALUES (?, ?, ?, ?, ?, ?)`,
			j.Title, j.Company, j.JobType, j.Requirements, j.Description, j.Deadline,
		)
		if err != nil {
			http.Error(w, "Insert error: "+err.Error(), 500)
			return
		}
		http.Redirect(w, r, "/jobs", http.StatusSeeOther)
		return
	}
	tmpl, err := template.ParseFiles("templates/add_job.html")
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), 500)
		return
	}
	tmpl.Execute(w, nil)
}

func editJob(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "Неверный id", 400)
		return
	}

	if r.Method == http.MethodPost {
		j := Job{
			ID:           id,
			Title:        r.FormValue("title"),
			Company:      r.FormValue("company"),
			JobType:      r.FormValue("job_type"),
			Requirements: r.FormValue("requirements"),
			Description:  r.FormValue("description"),
			Deadline:     r.FormValue("deadline"),
		}
		if j.Title == "" {
			http.Error(w, "Заполните обязательные поля", 400)
			return
		}
		_, err := db.Exec(`UPDATE jobs SET title=?, company=?, job_type=?, requirements=?, description=?, deadline=?
			WHERE id=?`,
			j.Title, j.Company, j.JobType, j.Requirements, j.Description, j.Deadline, j.ID,
		)
		if err != nil {
			http.Error(w, "Update error: "+err.Error(), 500)
			return
		}
		http.Redirect(w, r, "/jobs", http.StatusSeeOther)
		return
	}

	var j Job
	err = db.QueryRow(`SELECT id, title, COALESCE(company,''), COALESCE(job_type,''), COALESCE(requirements,''), COALESCE(description,''), COALESCE(deadline,''), created_at
		FROM jobs WHERE id=?`, id).
		Scan(&j.ID, &j.Title, &j.Company, &j.JobType, &j.Requirements, &j.Description, &j.Deadline, &j.CreatedAt)
	if err == sql.ErrNoRows {
		http.Error(w, "Вакансия не найдена", 404)
		return
	}
	if err != nil {
		http.Error(w, "DB error: "+err.Error(), 500)
		return
	}

	tmpl, err := template.ParseFiles("templates/edit_job.html")
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), 500)
		return
	}
	tmpl.Execute(w, j)
}

func deleteJob(w http.ResponseWriter, r *http.Request) {
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
	_, err = db.Exec("DELETE FROM jobs WHERE id=?", id)
	if err != nil {
		http.Error(w, "Delete error: "+err.Error(), 500)
		return
	}
	http.Redirect(w, r, "/jobs", http.StatusSeeOther)
}
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	
	// Importiert den reinen Go-Firebird-Treiber (unterstrich bedeutet: nur für die Treiber-Registrierung laden)
	_ "github.com/nakagami/firebirdsql"
)

// main ist der Einstiegspunkt des Go-Backend-Plugins.
// Es registriert das Plugin beim Grafana Plugin SDK, damit Grafana Anfragen an dieses Backend weiterleiten kann.
func main() {
	// Ersetzen Sie "your-org-firebird-datasource" exakt mit der ID aus Ihrer plugin.json
	err := datasource.Manage("your-org-firebird-datasource", NewFirebirdDataSource, datasource.ManageOpts{})
	if err != nil {
		log.DefaultLogger.Error(err.Error())
		os.Exit(1)
	}
}

// FirebirdDataSource repräsentiert die Instanz unserer Datenquelle.
// Hier könnten z. B. auch globale Konfigurationen oder Connection-Pools gehalten werden.
type FirebirdDataSource struct{}

// QueryModel definiert die Struktur, die vom Frontend (QueryEditor) als JSON an das Backend gesendet wird.
type QueryModel struct {
	RawSQL string `json:"rawSql"`
}

// SettingsModel spiegelt die unverschlüsselten Verbindungseinstellungen aus dem ConfigEditor wider.
type SettingsModel struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	User     string `json:"user"`
}

// NewFirebirdDataSource wird von Grafana aufgerufen, wenn eine neue Instanz der Datenquelle benötigt wird.
func NewFirebirdDataSource(setting backend.DataSourceInstanceSettings) (datasource.InstanceServicer, error) {
	return &FirebirdDataSource{}, nil
}

// getDB baut anhand der Plugin-Einstellungen und des sicheren Passworts eine Verbindung zur Firebird 2.5 Datenbank auf.
func (d *FirebirdDataSource) getDB(settings backend.DataSourceInstanceSettings) (*sql.DB, error) {
	var model SettingsModel
	
	// Parst die JSON-Daten, die der Nutzer im Grafana ConfigEditor eingegeben hat
	if err := json.Unmarshal(settings.JSONData, &model); err != nil {
		return nil, err
	}

	// Holt das sicher gespeicherte Passwort (secureJsonData) aus dem Grafana-Backend
	password := settings.DecryptedSecureJSONData["password"]
	
	port := model.Port
	if port == 0 {
		port = 3050 // Standard-Firebird-Port als Fallback
	}

	// DSN (Data Source Name) für den nakagami/firebirdsql Treiber zusammenbauen:
	// Format: user:password@host:port/pfad/zur/datenbank.fdb
	dsn := fmt.Sprintf("%s:%s@%s:%d/%s", model.User, password, model.Host, port, model.Database)
	
	// Öffnet die Verbindung (prüft syntaktisch, stellt aber evtl. noch keinen Sofort-Netzwerk-Socket her)
	db, err := sql.Open("firebirdsql", dsn)
	if err != nil {
		return nil, err
	}
	return db, nil
}

// CheckHealth wird ausgeführt, wenn der Nutzer im Grafana ConfigEditor auf den "Save & test"-Button klickt.
// Hier wird getestet, ob die Verbindung zur Datenbank tatsächlich funktioniert.
func (d *FirebirdDataSource) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	db, err := d.getDB(req.PluginContext.DataSourceInstanceSettings)
	if err != nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "Fehler beim Parsen der Konfiguration: " + err.Error(),
		}, nil
	}
	defer db.Close()

	// Prüft mit einem echten Ping zur Datenbank, ob der Server erreichbar ist und die Zugangsdaten stimmen
	if err := db.PingContext(ctx); err != nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "Verbindung zu Firebird fehlgeschlagen: " + err.Error(),
		}, nil
	}

	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: "Datenbankverbindung erfolgreich hergestellt!",
	}, nil
}

// QueryData ist das Herzstück: Es verarbeitet alle Abfragen (Queries) aus den Grafana-Dashboards/Panels.
func (d *FirebirdDataSource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	// Initialisiert das Antwortobjekt, das an Grafana zurückgeschickt wird
	response := backend.NewQueryDataResponse()

	// Verbindung für diesen Abfrage-Durchlauf öffnen
	db, err := d.getDB(req.PluginContext.DataSourceInstanceSettings)
	if err != nil {
		return response, err
	}
	defer db.Close()

	// Ein Dashboard kann mehrere Panels/Abfragen (Queries) gleichzeitig enthalten.
	// Deshalb iterieren wir durch alle übergebenen Queries.
	for _, q := range req.Queries {
		res := d.query(ctx, db, q)
		// Das Ergebnis wird anhand der eindeutigen RefID (z. B. "A", "B") des Panels gespeichert
		response.Responses[q.RefID] = res
	}

	return response, nil
}

// query führt eine einzelne SQL-Abfrage aus und mappt das Ergebnis in ein Grafana Data Frame.
func (d *FirebirdDataSource) query(ctx context.Context, db *sql.DB, query backend.DataQuery) backend.DataResponse {
	var response backend.DataResponse
	var model QueryModel

	// Liest das SQL-Statement aus dem Frontend-QueryEditor aus
	if err := json.Unmarshal(query.JSON, &model); err != nil {
		response.Error = err
		return response
	}

	if model.RawSQL == "" {
		return response // Leere Abfragen direkt ignorieren
	}

	// SQL-Abfrage an Firebird senden
	rows, err := db.QueryContext(ctx, model.RawSQL)
	if err != nil {
		response.Error = err
		return response
	}
	defer rows.Close()

	// Konvertiert die SQL-Zeilen in das von Grafana benötigte Data-Frame-Format
	frame, err := rowsToDataFrame(rows)
	if err != nil {
		response.Error = err
		return response
	}

	response.Frames = append(response.Frames, frame)
	return response
}

// rowsToDataFrame nimmt rohe SQL-Ergebniszeilen entgegen und wandelt sie dynamisch 
// in ein Grafana data.Frame um, damit Grafana Diagramme und Tabellen daraus zeichnen kann.
func rowsToDataFrame(rows *sql.Rows) (*data.Frame, error) {
	// Erstellt ein neues Data Frame mit dem Namen "response"
	frame := data.NewFrame("response")

	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	values := make([]interface{}, len(cols))
	valuePtrs := make([]interface{}, len(cols))
	for i := range cols {
		valuePtrs[i] = &values[i]
	}

	var rawRows [][]interface{}

	// Alle Zeilen aus der Datenbank einlesen
	for rows.Next() {
		err := rows.Scan(valuePtrs...)
		if err != nil {
			return nil, err
		}
		rowCopy := make([]interface{}, len(cols))
		for i, v := range values {
			rowCopy[i] = v
		}
		rawRows = append(rawRows, rowCopy)
	}

	// Spalten und Typen analysieren und in Grafana-Felder (Fields) mappen
	for i, colName := range cols {
		typeName := colTypes[i].DatabaseTypeName()
		
		// Beispielhafte Typ-Erkennung für Firebird 2.5 Zeitstempel
		if typeName == "TIMESTAMP" || typeName == "DATE" {
			timeSlice := make([]*time.Time, len(rawRows))
			for rIdx, row := range rawRows {
				if t, ok := row[i].(time.Time); ok {
					timeSlice[rIdx] = &t
				}
			}
			frame.Fields = append(frame.Fields, data.NewField(colName, nil, timeSlice))
		} else {
			// Standard-Fallback: Als Text/String behandeln
			strSlice := make([]string, len(rawRows))
			for rIdx, row := range rawRows {
				if row[i] != nil {
					strSlice[rIdx] = fmt.Sprintf("%v", row[i])
				} else {
					strSlice[rIdx] = ""
				}
			}
			frame.Fields = append(frame.Fields, data.NewField(colName, nil, strSlice))
		}
	}

	return frame, nil
}


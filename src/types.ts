import { DataSourceJsonData } from '@grafana/data';

// Öffentliche Konfigurationsdaten (ohne Passwörter)
export interface FirebirdDataSourceOptions extends DataSourceJsonData {
  host?: string;
  port?: number;
  database?: string; // z.B. /path/to/db.fdb oder localhost:3050:/path/to/db.fdb
  user?: string;
}

// Sensible Daten (Passwort)
export interface FirebirdSecureJsonData {
  password?: string;
}


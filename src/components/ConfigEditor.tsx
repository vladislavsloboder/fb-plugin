import React from 'react';
import { DataSourcePluginOptionsEditorProps } from '@grafana/data';
import { InlineField, Input, SecretInput } from '@grafana/ui';
import { FirebirdDataSourceOptions, FirebirdSecureJsonData } from '../types';

type Props = DataSourcePluginOptionsEditorProps<FirebirdDataSourceOptions, FirebirdSecureJsonData>;

export function ConfigEditor(props: Props) {
  const { options, onOptionsChange } = props;
  const { jsonData, secureJsonFields, secureJsonData } = options;

  // Handler für Änderungen an ungesicherten Feldern
  constonFieldChange = (key: keyof FirebirdDataSourceOptions) => (event: React.ChangeEvent<HTMLInputElement>) => {
    onOptionsChange({
      ...options,
      jsonData: {
        ...jsonData,
        [key]: event.target.value,
      },
    });
  };

  // Handler für Änderungen am Passwort
  const onPasswordChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    onOptionsChange({
      ...options,
      secureJsonData: {
        ...secureJsonData,
        password: event.target.value,
      },
    });
  };

  const onResetPassword = () => {
    onOptionsChange({
      ...options,
      secureJsonFields: {
        ...secureJsonFields,
        password: false,
      },
      secureJsonData: {
        ...secureJsonData,
        password: '',
      },
    });
  };

  return (
    <div className="gf-form-group">
      <h3>Firebird 2.5 Verbindungseinstellungen</h3>
      
      <InlineField label="Host" labelWidth={14}>
        <Input
          placeholder="localhost"
          value={jsonData.host || ''}
          onChange={onFieldChange('host')}
          width={30}
        />
      </InlineField>

      <InlineField label="Port" labelWidth={14}>
        <Input
          type="number"
          placeholder="3050"
          value={jsonData.port || 3050}
          onChange={onFieldChange('port')}
          width={30}
        />
      </InlineField>

      <InlineField label="Datenbankpfad" labelWidth={14}>
        <Input
          placeholder="/var/lib/firebird/2.5/data/mydb.fdb"
          value={jsonData.database || ''}
          onChange={onFieldChange('database')}
          width={40}
        />
      </InlineField>

      <InlineField label="User" labelWidth={14}>
        <Input
          placeholder="SYSDBA"
          value={jsonData.user || ''}
          onChange={onFieldChange('user')}
          width={30}
        />
      </InlineField>

      <InlineField label="Passwort" labelWidth={14}>
        <SecretInput
          isConfigured={Boolean(secureJsonFields?.password)}
          value={secureJsonData?.password || ''}
          placeholder="Passwort eingeben"
          width={30}
          onChange={onPasswordChange}
          onReset={onResetPassword}
        />
      </InlineField>
    </div>
  );
}


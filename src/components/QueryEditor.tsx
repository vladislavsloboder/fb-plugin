import React, { ChangeEvent } from 'react';
import { InlineField, Input } from '@grafana/ui';
import { QueryEditorProps } from '@grafana/data';
import { DataSource } from '../datasource';
import { MyDataSourceOptions, MyQuery } from '../types';

type Props = QueryEditorProps<DataSource, MyQuery, MyDataSourceOptions>;

export function QueryEditor({ query, onChange, onRun }: Props) {
  const onSqlChange = (event: ChangeEvent<HTMLInputElement>) => {
    onChange({ ...query, rawSql: event.target.value });
    onRun();
  };

  const { rawSql } = query;

  return (
    <div className="gf-form-group">
      <InlineField label="SQL Query" labelWidth={12} grow>
        <Input
          value={rawSql || ''}
          onChange={onSqlChange}
          placeholder="SELECT * FROM MY_TABLE"
        />
      </InlineField>
    </div>
  );
}

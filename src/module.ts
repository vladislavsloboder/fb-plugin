import { DataSourcePlugin } from '@grafana/data';
import { DataSource } from './datasource';
import { MyQuery, MyDataSourceOptions } from './types';
import { ConfigEditor } from './components/ConfigEditor';
import { QueryEditor } from './components/QueryEditor';

export const plugin = new DataSourcePlugin<DataSource, MyQuery, MyDataSourceOptions>(DataSource)
  .setConfigEditor(ConfigEditor)
  .setQueryEditor(QueryEditor);


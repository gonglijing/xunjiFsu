import * as alarms from './alarms';
import * as auth from './auth';
import * as collector from './collector';
import * as data from './data';
import * as devices from './devices';
import * as drivers from './drivers';
import * as gateway from './gateway';
import * as metrics from './metrics';
import * as northbound from './northbound';
import * as resources from './resources';
import * as status from './status';
import * as storage from './storage';
import * as thresholds from './thresholds';

const api = {
  alarms,
  auth,
  collector,
  data,
  devices,
  drivers,
  gateway,
  metrics,
  northbound,
  resources,
  status,
  storage,
  thresholds,
};

export default api;

export {
  alarms as alarmsAPI,
  auth as authAPI,
  collector as collectorAPI,
  data as dataAPI,
  devices as devicesAPI,
  drivers as driversAPI,
  gateway as gatewayAPI,
  metrics as metricsAPI,
  northbound as northboundAPI,
  resources as resourcesAPI,
  status as statusAPI,
  storage as storageAPI,
  thresholds as thresholdsAPI,
};

import http from "k6/http";

export const DatasourcesEndpoint = class DatasourcesEndpoint {
  constructor(httpClient) {
    this.httpClient = httpClient;
  }

  getAll() {
    return this.httpClient.get('/datasources');
  }

  getById(id) {
    return this.httpClient.get(`/datasources/${id}`);
  }

  getByName(name) {
    return this.httpClient.get(`/datasources/name/${name}`);
  }

  create(payload) {
    return this.httpClient.post(`/datasources`, JSON.stringify(payload));
  }

  update(id, payload) {
    return this.httpClient.put(`/datasources/${id}`, JSON.stringify(payload));
  }
};

export const DashboardsEndpoint = class DashboardsEndpoint {
  constructor(httpClient) {
    this.httpClient = httpClient;
  }

  getAll() {
    return this.httpClient.get('/dashboards');
  }

  upsert(payload) {
    return this.httpClient.post(`/dashboards/db`, JSON.stringify(payload));
  }
};

export const OrganizationsEndpoint = class OrganizationsEndpoint {
  constructor(httpClient) {
    this.httpClient = httpClient;
  }

  getById(id) {
    return this.httpClient.get(`/orgs/${id}`);
  }

  getByName(name) {
    return this.httpClient.get(`/orgs/name/${name}`);
  }

  create(name) {
    let payload = {
      name: name,
    };
    return this.httpClient.post(`/orgs`, JSON.stringify(payload));
  }
};

export const UIEndpoint = class UIEndpoint {
  constructor(httpClient) {
    this.httpClient = httpClient;
  }

  login(username, pwd) {
    const payload = { user: username, password: pwd };
    return this.httpClient.formPost('/login', payload);
  }

  renderPanel(orgId, dashboardUid, panelId) {
    return this.httpClient.get(
      `/render/d-solo/${dashboardUid}/graph-panel`,
      {
        orgId,
        panelId,
        width: 1000,
        height: 500,
        tz: 'Europe/Stockholm',
      }
    );
  }
}

export const GrafanaClient = class GrafanaClient {
  constructor(httpClient) {
    httpClient.onBeforeRequest = (params) => {
      if (this.orgId && this.orgId > 0) {
        params.headers = params.headers || {};
        params.headers["X-Grafana-Org-Id"] = this.orgId;
      }
    }

    this.raw = httpClient;
    this.dashboards = new DashboardsEndpoint(httpClient.withUrl('/api'));
    this.datasources = new DatasourcesEndpoint(httpClient.withUrl('/api'));
    this.orgs = new OrganizationsEndpoint(httpClient.withUrl('/api'));
    this.ui = new UIEndpoint(httpClient);
  }

  loadCookies(cookies) {
    for (let [name, value] of Object.entries(cookies)) {
      http.cookieJar().set(this.raw.url, name, value);
    }
  }

  saveCookies() {
    return http.cookieJar().cookiesForURL(this.raw.url + '/');
  }

  batch(requests) {
    return this.raw.batch(requests);
  }

  withOrgId(orgId) {
    this.orgId = orgId;
  }
}

export const BaseClient = class BaseClient {
  constructor(url, subUrl) {
    if (url.endsWith('/')) {
      url = url.substring(0, url.length - 1);
    }

    if (subUrl.endsWith('/')) {
      subUrl = subUrl.substring(0, subUrl.length - 1);
    }

    this.url = url + subUrl;
    this.onBeforeRequest = () => {};
  }

  withUrl(subUrl) {
    let c = new BaseClient(this.url,  subUrl);
    c.onBeforeRequest = this.onBeforeRequest;
    return c;
  }

  beforeRequest(params) {

  }

  get(url, queryParams, params) {
    params = params || {};
    this.onBeforeRequest(params);

    if (queryParams) {
      url += '?' + Array.from(Object.entries(queryParams)).map(([key, value]) =>
        `${key}=${encodeURIComponent(value)}`
      ).join('&');
    }

    return http.get(this.url + url, params);
  }

  formPost(url, body, params) {
    params = params || {};
    this.beforeRequest(params);
    this.onBeforeRequest(params);
    return http.post(this.url + url, body, params);
  }

  post(url, body, params) {
    params = params || {};
    params.headers = params.headers || {};
    params.headers['Content-Type'] = 'application/json';

    this.beforeRequest(params);
    this.onBeforeRequest(params);
    return http.post(this.url + url, body, params);
  }

  put(url, body, params) {
    params = params || {};
    params.headers = params.headers || {};
    params.headers['Content-Type'] = 'application/json';

    this.onBeforeRequest(params);
    return http.put(this.url + url, body, params);
  }

  delete(url, params) {
    params = params || {};
    this.beforeRequest(params);
    this.onBeforeRequest(params);
    return http.del(this.url + url, null, params);
  }

  batch(requests) {
    for (let n = 0; n < requests.length; n++) {
      let params = requests[n].params || {};
      params.headers = params.headers || {};
      params.headers['Content-Type'] = 'application/json';
      this.beforeRequest(params);
      this.onBeforeRequest(params);
      requests[n].params = params;
      requests[n].url = this.url + requests[n].url;
      if (requests[n].body) {
        requests[n].body = JSON.stringify(requests[n].body);
      }
    }

    return http.batch(requests);
  }
}

export const createClient = (url) => {
  return new GrafanaClient(new BaseClient(url, ''));
}

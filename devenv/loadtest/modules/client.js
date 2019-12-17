import http from "k6/http";

export const UIEndpoint = class UIEndpoint {
  constructor(httpClient) {
    this.httpClient = httpClient;
  }

  login(username, pwd) {
    const payload = { user: username, password: pwd };
    return this.httpClient.formPost('/login', payload);
  }

  render() {
    return this.httpClient.get('/render/d-solo/_CPokraWz/graph-panel?orgId=1&panelId=1&width=1000&height=500&tz=Europe%2FStockholm')
  }
}

export const GrafanaClient = class GrafanaClient {
  constructor(httpClient) {
    httpClient.onBeforeRequest = this.onBeforeRequest;
    this.raw = httpClient;
    this.ui = new UIEndpoint(httpClient);
  }

  batch(requests) {
    return this.raw.batch(requests);
  }

  withOrgId(orgId) {
    this.orgId = orgId;
  }

  onBeforeRequest(params) {
    if (this.orgId && this.orgId > 0) {
      params = params.headers || {};
      params.headers["X-Grafana-Org-Id"] = this.orgId;
    }
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

  get(url, params) {
    params = params || {};
    this.beforeRequest(params);
    this.onBeforeRequest(params);
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

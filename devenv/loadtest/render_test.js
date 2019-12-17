import { check, group } from 'k6';
import { createClient } from './modules/client.js';

export let options = {
  noCookiesReset: true
};

let endpoint = __ENV.URL || 'http://localhost:3000';
const client = createClient(endpoint);

export default () => {
  group("render test", () => {
    if (__ITER === 0) {
      group("user authenticates thru ui with username and password", () => {
        let res = client.ui.login('admin', 'admin');

        check(res, {
          'response status is 200': (r) => r.status === 200,
        });
      });
    }

    if (__ITER !== 0) {
      group("render graph panel", () => {
        const response = client.ui.render();
        check(response, {
          'response status is 200': (r) => r.status === 200,
        });
      });
    }
  });
}

export const teardown = () => {}

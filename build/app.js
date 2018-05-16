"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const grpc = require("grpc");
const SERVER_ADDRESS = '127.0.0.1:50051';
const RENDERER_PROTO_PATH = __dirname + '/../proto/renderer.proto';
const GRPC_HEALTH_PROTO_PATH = __dirname + '/../proto/health.proto';
exports.RENDERER_PROTO = grpc.load(RENDERER_PROTO_PATH).models;
exports.GRPC_HEALTH_PROTO = grpc.load(GRPC_HEALTH_PROTO_PATH).grpc.health.v1;
/**
 * Implements the Health Check RPC method.
 */
function check(call, callback) {
    callback(null, { status: 'SERVING' });
}
function render(call, callback) {
    console.log('render', call.request);
    callback(null, { filePath: call.request.url + 'resp' });
}
/**
 * Starts an RPC server that receives requests for the Greeter service at the
 * sample server port
 */
function main() {
    var server = new grpc.Server();
    server.addService(exports.GRPC_HEALTH_PROTO.Health.service, { check: check });
    server.addService(exports.RENDERER_PROTO.Renderer.service, { render: render });
    server.bind(SERVER_ADDRESS, grpc.ServerCredentials.createInsecure());
    server.start();
    console.log(`1|1|tcp|${SERVER_ADDRESS}|grpc`);
    console.error('Renderer plugin started');
}
main();
// const puppeteer = require('puppeteer');
//
// var argv = require('minimist')(process.argv.slice(2));
// console.dir(argv);
//
// (async () => {
//   const browser = await puppeteer.launch();
//   const page = await browser.newPage();
//   await page.goto('http://localhost:3000');
//   await page.screenshot({path: 'example.png'});
//
//   await browser.close();
// })();
//# sourceMappingURL=app.js.map
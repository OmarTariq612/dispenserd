const express = require("express");
const fs = require("fs");
const bodyParser = require("body-parser");
const mraa = require("mraa");

function init() {
    const pin3 = new mraa.Gpio(3);
    const pin5 = new mraa.Gpio(5);
    const pin6 = new mraa.Gpio(6);
    pin3.dir(mraa.DIR_OUT);
    pin5.dir(mraa.DIR_OUT);
    pin6.dir(mraa.DIR_OUT);
    pin3.wirte(0);
    pin5.wirte(0);
    pin6.wirte(0);
}

const app = express();

if (process.argv.length !== 3) {
    console.error("unix socket path is required");
    process.exit(1);
}

const unixSocketPath = process.argv[2];

app.use(bodyParser.json());

const pin = new mraa.Gpio(3);

app.post("/notify", (_, res) => {
    console.log("notify");
    pin.write(1);
    setTimeout(() => pin.write(0), 800);
    res.status(200).end();
});

app.post("/resetTo", (req, res) => {
    console.log(`reset to ${req.body.duration} seconds`)
    res.status(200).end();
});

function ListenAndServe(app, unixSocketPath) {
    app.listen(unixSocketPath, () => {
        console.log(`Listening on ${unixSocketPath}`);
    }).on('error', (error) => {
        if (error.syscall !== 'listen') {
            throw error;
        }

        switch (error.code) {
            case 'EACCES':
                console.error(`"${unixSocketPath}" requires elevated privileges`);
                process.exit(1);
            case 'EADDRINUSE':
                console.error(`"${unixSocketPath}" is already in use`);
                console.error(`removing the unix socket file: "${unixSocketPath}"`);
                fs.unlinkSync(unixSocketPath);
                setImmediate(() => {
                    ListenAndServe(app, unixSocketPath);
                });
                break;
            default:
                throw error;
        }
    });
}

init();
ListenAndServe(app, unixSocketPath);

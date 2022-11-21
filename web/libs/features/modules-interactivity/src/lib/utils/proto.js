/* eslint-disable no-underscore-dangle */
import * as pb from 'protobufjs/light';

import EventEmitter from 'events';

let DEBUG = false;

class Events extends EventEmitter {}

const debug = (...args) => {
    // eslint-disable-next-line no-console
    if (DEBUG) console.log(...args);
};

class PublicAPI {
    constructor(vxapi) {
        this.vxapi = vxapi;
    }

    subscribe(f, type = '*') {
        debug(`SUBSCRIBE type= ${type}`);
        switch (type) {
            case '*':
                this.vxapi._userHandlers = { text: f, data: f, file: f, msg: f, act: f };
                break;
            case 'data':
                this.vxapi._userHandlers = { data: f };
                break;
            case 'text':
                this.vxapi._userHandlers = { text: f };
                break;
            case 'file':
                this.vxapi._userHandlers = { file: f };
                break;
            case 'msg':
                this.vxapi._userHandlers = { msg: f };
                break;
            case 'action':
                this.vxapi._userHandlers = { act: f };
                break;
        }
    }

    getState() {
        return this.vxapi._state;
    }

    sendData(data) {
        debug(`sendData: ${data}`);
        const m = {
            type: 'data',
            data
        };
        this.vxapi.send(m);
    }

    sendText(text) {
        debug(`sendText: ${text}`);
        const m = {
            type: 'text',
            data: text
        };
        this.vxapi.send(m);
    }

    sendFile(data, name, uniq, number, total) {
        debug(`sendFile: ${data} | ${name} | ${uniq} | ${number} | ${total}`);
        const m = {
            type: 'file',
            data,
            name,
            number,
            total,
            uniq
        };
        this.vxapi.send(m);
    }

    sendMsg(data, mtype) {
        debug(`sendMsg: ${data} | ${mtype}`);
        const m = {
            type: 'msg',
            data,
            mtype
        };
        this.vxapi.send(m);
    }

    sendAction(data, name) {
        debug(`sendAction: ${data} | ${name}`);
        const m = {
            type: 'action',
            data,
            name
        };
        this.vxapi.send(m);
    }
}

export class VXAPI {
    constructor(params) {
        this._module = params.moduleName;
        this._srcToken = '';
        this._dstToken = '';

        this.events = new Events();
        this._queue = [];

        this._agentProto = params.agentProto;

        this.pb = pb;
        this.pp = params.protocolProto.lookupType('protocol.Packet');
        this.pcc = params.protocolProto.lookupType('protocol.Packet.Content');
        this.endpoint = `${params.hostPort}/api/v1/vxpws/browser/${params.agentHash}/`;
        this._socket = null;
        this._state = 0; // init state of socket connection
        this.publicAPI = new PublicAPI(this);
        this.events.on('stateChanged', () => this.changedState());
    }

    changedState() {
        const self = this;
        if (self._state === 2) {
            if (self._queue.length > 0) {
                debug('queue before: ', self._queue.length);
                for (let i = 0; i < self._queue.length; i++) {
                    const _packet = self._queue.pop();
                    _packet.source = this._srcToken;
                    _packet.destination = this._dstToken;

                    const err = self.pp.verify(_packet);
                    if (err) {
                        // eslint-disable-next-line no-console
                        console.log(`queue: failed to make _packet: ${err}`);
                        continue;
                    }

                    const packet = self.pp.create(_packet);
                    const buffer = self.pp.encode(packet).finish();
                    self._socket.send(buffer);
                }
                debug('queue after: ', self._queue.length);
                setTimeout(() => {
                    self.changedState();
                }, 1000);
            }
        }
    }

    connect() {
        const self = this;

        return new Promise(function (resolve, reject) {
            const socket = new WebSocket(self.endpoint); // get endpoint from private field
            socket.onopen = function () {
                self._socket = socket;
                self._handshake();
                const pingDelay = 5000;
                const pingBuffer = new Uint8Array([80, 73, 78, 71]);
                const sendPing = () => {
                    if ([0, 1].includes(socket.readyState)) {
                        if (self._state === 2) socket.send(pingBuffer.buffer);
                        debug(`send: ping`);
                        setTimeout(sendPing, pingDelay);
                    } else {
                        debug('ping sender on connection was closed', socket.readyState);
                    }
                };
                setTimeout(sendPing, pingDelay);
                resolve(self.publicAPI);
            };
            socket.onclose = function (event) {
                if (event.wasClean || self._state === 3) {
                    debug('Connection was closed correctly: ', event.wasClean, self._state);
                    self._state = 4; // closed state
                } else {
                    debug('Dirty connection closed: ', self._state);
                    debug(`Return code: ${event.code} reason: ${event.reason}`);
                    setTimeout(() => {
                        self.connect();
                    }, 1000);
                }
            };
            socket.onerror = function (error) {
                reject(error);
            };
            socket.onmessage = function (e) {
                self._msgHandler(e);
            };
        });
    }

    close() {
        debug('run close function on socket');
        if (this._socket) {
            this._state = 3; // closing state
            this._socket.close();
        }
    }

    send(msg) {
        const _packet = {
            module: this._module,
            source: this._srcToken,
            destination: this._dstToken,
            timestamp: Math.round(new Date().getTime() / 1000)
        };
        let _content = {};
        debug(`Packet header: ${JSON.stringify(_packet)}`);
        switch (msg.type) {
            case 'data':
                debug(`Pack content: ${msg.type}, ${msg.data}`);
                _content = {
                    type: this.pcc.Type.DATA,
                    data: new TextEncoder('utf-8').encode(msg.data)
                };
                break;
            case 'text':
                _content = {
                    type: this.pcc.Type.TEXT,
                    data: msg.data
                };
                break;
            case 'file':
                const _part = {
                    number: msg.number,
                    total: msg.total
                };
                const error = this.pcc.Part.verify(_part);
                if (error) throw Error(error); // check error

                _content = {
                    type: this.pcc.Type.FILE,
                    data: msg.data,
                    name: msg.name,
                    part: this.pcc.Part.create(_part),
                    uniq: msg.uniq
                };
                break;
            case 'msg':
                let mtype;
                switch (msg.mtype) {
                    case 'error':
                        mtype = this.pcc.MsgType.ERROR;
                        break;
                    case 'warn':
                        mtype = this.pcc.MsgType.WARNING;
                        break;
                    case 'info':
                        mtype = this.pcc.MsgType.INFO;
                        break;
                    case 'debug':
                        mtype = this.pcc.MsgType.DEBUG;
                        break;
                    default:
                        throw Error('unknown message type');
                }
                _content = {
                    type: this.pcc.Type.MSG,
                    data: new TextEncoder('utf-8').encode(msg.data),
                    msg_type: mtype
                };
                break;
            case 'action':
                _content = {
                    type: this.pcc.Type.ACT,
                    data: new TextEncoder('utf-8').encode(msg.data),
                    name: msg.name
                };
                break;
        }

        let err = this.pcc.verify(_content);
        if (err) throw Error(err); // check error
        _packet.content = this.pcc.create(_content);

        err = this.pp.verify(_packet);
        if (err) throw Error(err); // check error

        if (this._state !== 2) {
            this._queue.push(_packet); // add message to queue
            if (this._state === 4) {
                this._state = 0;
                this.connect();
            }
            debug('send: try to reconnect after the method was called');
        } else {
            const packet = this.pp.create(_packet);
            const buffer = this.pp.encode(packet).finish();
            this._socket.send(buffer);
        }
    }

    setDebug(value) {
        DEBUG = value;
    }

    _msgHandler(msg) {
        const data = msg.data;
        const self = this;
        const fileReader = new FileReader();

        switch (self._state) {
            case 0: // waiting for auth request...
                break;
            case 1: // waiting for auth response...
                try {
                    fileReader.onload = function (event) {
                        const authenticationResponse = self._agentProto.lookupType('agent.AuthenticationResponse');
                        const authenticationResponseMsg = authenticationResponse.decode(
                            new Uint8Array(event.target.result)
                        );
                        if (authenticationResponseMsg.status !== 'authorized') {
                            self._state = 4; // closed state
                            self.events.emit('stateChanged');

                            return;
                        }
                        self._srcToken = authenticationResponseMsg.atoken;
                        self._dstToken = authenticationResponseMsg.stoken;
                        debug(`RECV HS RESPONSE ${JSON.stringify(authenticationResponseMsg)}`);
                        self._state = 2; // connected
                        self.events.emit('stateChanged');
                    };

                    fileReader.onerror = function (e) {
                        // eslint-disable-next-line no-console
                        console.log(e);
                        self._state = 0;
                    };
                    fileReader.readAsArrayBuffer(data);
                } catch (e) {
                    if (e instanceof pb.util.ProtocolError) {
                        // eslint-disable-next-line no-console
                        console.log(e);
                    } else {
                        debug('invalid format', e);
                    }
                }
                break;
            case 2: // auth ok, recv packet
                fileReader.onload = function (event) {
                    const decodedMessage = self.pp.decode(new Uint8Array(event.target.result));
                    const obj = self.pp.toObject(decodedMessage);
                    if (self._userHandlers !== undefined) {
                        switch (obj.content.type) {
                            case self.pcc.Type.DATA:
                                if (self._userHandlers.data !== undefined) {
                                    self._userHandlers.data(obj);
                                }
                                break;
                            case self.pcc.Type.TEXT:
                                if (self._userHandlers.text !== undefined) {
                                    self._userHandlers.text(obj);
                                }
                                break;
                            case self.pcc.Type.FILE:
                                if (self._userHandlers.file !== undefined) {
                                    self._userHandlers.file(obj);
                                }
                                break;
                            case self.pcc.Type.MSG:
                                if (self._userHandlers.msg !== undefined) {
                                    self._userHandlers.msg(obj);
                                }
                                break;
                            case self.pcc.Type.ACT:
                                if (self._userHandlers.act !== undefined) {
                                    self._userHandlers.act(obj);
                                }
                                break;
                        }
                    }
                };

                fileReader.onerror = function (e) {
                    // eslint-disable-next-line no-console
                    console.log(e);
                };
                fileReader.readAsArrayBuffer(data);
                break;
            default:
                debug('invalid state', self._state);
        }
    }

    _handshake() {
        const authenticationRequest = this._agentProto.lookupType('agent.AuthenticationRequest');
        const ts = Math.round(new Date().getTime() / 1000);
        const message = authenticationRequest.create({
            timestamp: ts,
            atoken: this._srcToken,
            aversion: 'v1.0.0'
        });
        const buffer = authenticationRequest.encode(message).finish();
        this._state = 1;
        this._socket.send(buffer);
    }
}

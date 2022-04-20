/*
 * good ol' global variables
 */
var playing = false;

// ui
var playstop = document.querySelector("#playstop");
var info = document.querySelector("#info");

// websocket
var ws;

// audio
var audioContext = new AudioContext();
var source, gainNode, scriptNode;

// buffer from websocket to audio
var buf = [];
var bufInterval;

function notify(msg) {
    info.innerHTML = msg;
    // console.log("info:", msg);
}

/*
 * websocket to get samples from server
 */

function startSocket() {
    if (ws) {
        return false;
    }

    var proto = "ws://"
    if (window.location.protocol == "https:") {
        proto = "wss://"
    }

    ws = new WebSocket(proto + window.location.host + "/websocket");

    // ws.onopen = function(evt) {
    //     notify("websocket open");
    // }

    ws.onmessage = function(evt) {
        // shove incoming samples into our buffer
        samples = evt.data.split(",");
        buf = buf.concat(samples);
    }

    ws.onclose = function(evt) {
        if (!evt.wasClean) {
            console.log("onclose:", evt);
            notify("❓ websocket error ❓");
            stop();
        }
        stopAudio();
        ws = null;
    }

    ws.onerror = function(evt) {
        console.log("websocket error:", evt);
    }

}

function stopSocket() {
    if (ws) {
        ws.close();
    }   
}

/*
 * audio stuff to play sound
 */

function bufferAndStartAudio() {
    var toBuffer = audioContext.sampleRate;
    // var toBuffer = audioContext.sampleRate / 8;
    var bufInterval = null;
    bufAndStart = function() {
        var pct = Math.floor(buf.length / toBuffer * 100);
        console.log("✨ buffering (" + pct + "%) ✨");
        notify("✨ b u f f e r i n g ✨");

        if (buf.length > toBuffer) {
            // notify("🎶 playing 🎶");
            notify("");
            startAudio();
            clearInterval(bufInterval);
        }
    }
    bufInterval = setInterval(bufAndStart, 100);
}

function startAudio() {
    if (source) {
        return false;
    }

    // empty constant audio source
    source = audioContext.createConstantSource();

    // make the sounds louder
    gainNode = audioContext.createGain();
    gainNode.gain.value = 2.5; // ok number?

    // script processor is technically deprecated, but still supported by everyone (except IE lol)
    // and its replacement "audio worklet" doesn't seem to be natively supported by chrome? soooo...
    // https://developer.mozilla.org/en-US/docs/Web/API/BaseAudioContext/createScriptProcessor#example
    scriptNode = audioContext.createScriptProcessor(0, 0, 1);
    // scriptNode = audioContext.createScriptProcessor(4096, 0, 1); // 16384 is max, must be power of 2
    bufferSize = scriptNode.bufferSize;
    // console.log("buferSize:", bufferSize);

    // hook the bits together
    source.connect(scriptNode);
    scriptNode.connect(gainNode);
    gainNode.connect(audioContext.destination);

    // continuously cram our incoming samples into the audio stream
    scriptNode.onaudioprocess = function(evt) {
        // console.log(evt);
        var outputBuffer = evt.outputBuffer;

        bufLen = buf.length;
        if (bufLen < bufferSize) {
            console.log("buf is kinda small:", bufLen, "/", bufferSize);
        }

        // some named inline functions are mainly so performance profiling reads easier :P
        writeOut = function(channel) {
            var out = outputBuffer.getChannelData(channel);
            var size = out.length;
            var lilBuf = buf.slice(0, size);
            buf = buf.slice(size); // mutants are cool. // TODO: hiiii can this cause low tones somehow?
            writeToArray = function(val, idx, _a) {
                out[idx] = val;
            }
            lilBuf.forEach(writeToArray);
            // lilBuf.forEach(function(v, i, _a) {
            //     out[i] = v;
            // });
        }

        // TODO: stereo
        for (var channel = 0; channel < outputBuffer.numberOfChannels; channel++) {
            writeOut(channel);
        }
    }
    source.start();
}

function stopAudio() {
    if (source) {
        source.stop();
        source.disconnect(scriptNode);
        scriptNode.disconnect(gainNode);
        gainNode.disconnect(audioContext.destination);
    }
    source = null;
    buf = [];
    clearInterval(bufInterval);
}

/*
 * play/stop buttons
 */

function play() {
    startSocket();
    bufferAndStartAudio();

    playing = true;
    playstop.innerHTML = "🎶 stop 🎶";
    playstop.className = "stop";
}

function stop() {
    stopAudio();
    stopSocket();

    playing = false;
    playstop.innerHTML = "🎶 play 🎶";
    playstop.className = "play";
}

playstop.onclick = function() {
    if (playing) {
        stop();
    } else {
        play();
    }
}

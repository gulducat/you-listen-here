// raw xhr is apparently the simplest way to get a streaming http connection..?
// fetch() isn't good for it, socket.io seems mixed and more complicated.
// todo: maybe actual websocket is the way to go?  but for now...
var xhr = new XMLHttpRequest();

// set up us a web audio api
const audioCtx = new (window.AudioContext || window.webkitAudioContext)();
console.log("sampleRate: " + audioCtx.sampleRate);
bufferSize = audioCtx.sampleRate;
var buffer, chData, source;

var lastSpot = 0;
var buf = [];

async function makeSound(samples) {
    // each source can only be sent once, so we do this over every time here.
    // the order of these things is important, but i'm not totally clear on why...
    source = audioCtx.createBufferSource();

    // with compressor
    // var compressor = audioCtx.createDynamicsCompressor();
    // compressor.threshold.value = -40;
    // compressor.ratio.value = 20;
    // compressor.reduction.value = -20;
    // source.connect(compressor);
    // compressor.connect(audioCtx.destination);
    // without compressor
    source.connect(audioCtx.destination);

    source.buffer = buffer;
    buffer = audioCtx.createBuffer(
        1,
        samples.length,
        audioCtx.sampleRate,
    );
    chData = buffer.getChannelData(0);

    // put em in the channel
    // floats = new Float32Array(samples)
    samples.forEach((samp, i, _a) => {
        chData[i] = samp;
    })

    console.log("start -- samples:", samples.length, "; buf:", buf.length);
    source.start();
}

xhr.onprogress = function () {
    // some acrobatics to break the stream up into sensible chunks, since it is cumulative over time.
    // todo: this could add up in browser memory, do occasional dis-and-re-connect?
    // todo: or just bite the bullet and do websockets, probably should just do that.
    txt = xhr.responseText;
    txt = txt.slice(lastSpot);
    lastSpot = xhr.responseText.length;

    lines = txt.split("\n"); // maybe incomplete parts? idk
    lines.forEach((line, _idx, _arr) => {
        samples = line.split(",");
        buf = buf.concat(samples);
        if (buf.length >= bufferSize) {
            samps = buf.slice(0, bufferSize);
            buf = buf.slice(bufferSize); // reset our lil sample buffer.
            makeSound(samps);
        }
    });
};

xhr.onreadystatechange = function () {
    // console.log("readyState:", xhr.readyState)
    if (xhr.readyState === 4) {
        console.log("xhr.exit", xhr.status, xhr.responseText.length);
    }
};

var play = document.querySelector('.play');
var stop = document.querySelector('.stop');

play.onclick = function() {
    buf = [];
    console.log("xhr.send");
    xhr.open("GET", "/stream");
    xhr.send();
}
stop.onclick = function() {
    xhr.abort();
}

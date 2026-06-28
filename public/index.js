const PROTOCOL = window.location.protocol === "https:" ? "wss:" : "ws:";
const SOCKET_CONNECTION_URL = `${PROTOCOL}//${window.location.host}/ws`;

const RTC_CONFIG = {
  iceCandidatePoolSize: 10,
  iceServers: [
    {
      urls: ["stun:stun1.l.google.com:19302", "stun:stun2.l.google.com:19302"],
    },
    {
      urls: "turn:your-turn-host:3478",
      username: "user",
      credential: "pass",
    },
  ],
};

let rtc;
let socket, peerId;
let localStream, remoteStream;
let ready = false;
let rejected = false;
let messageQueue = [];

function createRTC() {
  if (rtc) {
    rtc.onicecandidate = null;
    rtc.oniceconnectionstatechange = null;
    rtc.ontrack = null;
    rtc.close();
  }

  rtc = new RTCPeerConnection(RTC_CONFIG);

  rtc.onicecandidate = (event) => {
    if (event.candidate && peerId) {
      socket.send(
        JSON.stringify({
          type: "ICE_CANDIDATE",
          userId: peerId,
          data: event.candidate,
        }),
      );
    }
  };

  rtc.oniceconnectionstatechange = () => {
    console.log("ICE state:", rtc.iceConnectionState);
  };

  const remoteVideoElement = document.querySelector(".stream__remote");

  remoteStream = new MediaStream();
  remoteVideoElement.srcObject = remoteStream;

  rtc.ontrack = (event) => {
    event.streams[0].getTracks().forEach((track) => {
      remoteStream.addTrack(track);
    });
    remoteVideoElement.hidden = false;
    document.querySelector(".streams").classList.add("streams--active");
  };

  if (localStream) {
    localStream.getTracks().forEach((track) => {
      rtc.addTrack(track, localStream);
    });
  }
}

window.addEventListener("load", async () => {
  try {
    socket = new WebSocket(SOCKET_CONNECTION_URL);

    socket.onopen = () => console.log("Socket connected");

    async function handleMessage(data) {
      if (data.type === "START_CALL") {
        peerId = data.data;

        const offer = await rtc.createOffer();
        await rtc.setLocalDescription(offer);

        socket.send(
          JSON.stringify({
            type: "PEER_CONNECTION_REQUEST",
            userId: peerId,
            data: offer,
          }),
        );
      } else if (data.type === "PEER_CONNECTION_REQUEST") {
        peerId = data.userId;

        await rtc.setRemoteDescription(data.data);

        const answer = await rtc.createAnswer();
        await rtc.setLocalDescription(answer);

        socket.send(
          JSON.stringify({
            type: "PEER_CONNECTION_RESPONSE",
            userId: peerId,
            data: answer,
          }),
        );
      } else if (data.type === "PEER_CONNECTION_RESPONSE") {
        peerId = data.userId;

        if (!rtc.currentRemoteDescription) {
          await rtc.setRemoteDescription(data.data);
        }
      } else if (data.type === "ICE_CANDIDATE") {
        try {
          await rtc.addIceCandidate(data.data);
        } catch (err) {
          console.error("Error adding ICE candidate:", err);
        }
      } else if (data.type === "PEER_DISCONNECTED") {
        peerId = null;
        document.querySelector(".stream__remote").hidden = true;
        document.querySelector(".streams").classList.remove("streams--active");
        createRTC();
      }
    }

    socket.onmessage = async (e) => {
      const data = JSON.parse(e.data);

      if (data.type === "ERROR") {
        rejected = true;
        document.querySelector(".streams").hidden = true;
        document.querySelector(".error-message").hidden = false;
        if (localStream) {
          localStream.getTracks().forEach((t) => t.stop());
        }
        return;
      }

      if (!ready) {
        messageQueue.push(data);
        return;
      }

      await handleMessage(data);
    };

    socket.onclose = () => console.log("Socket closed");
    socket.onerror = () => {
      console.log("Socket error, closing connection");
      socket.close();
    };

    if (rejected) return;

    localStream = await navigator.mediaDevices.getUserMedia({
      video: true,
      audio: true,
    });

    if (rejected) {
      localStream.getTracks().forEach((t) => t.stop());
      return;
    }

    const localVideoElement = document.querySelector(".stream__local");
    localVideoElement.srcObject = localStream;
    localVideoElement.hidden = false;

    createRTC();

    ready = true;
    for (const msg of messageQueue) {
      await handleMessage(msg);
    }
    messageQueue = [];
  } catch (err) {
    console.error(err);
    alert(err);
  }
});

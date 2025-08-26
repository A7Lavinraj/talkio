const SOCKET_CONNECTION_URL = "ws://localhost:8080/ws";

const rtc = new RTCPeerConnection({
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
});

class WebSocketSingleton {
  constructor(url) {
    if (WebSocketSingleton.instance) return WebSocketSingleton.instance;

    this.url = url;
    this.socket = new WebSocket(url);
    if (!this.socket) throw new Error("Unable to create Socket");

    this.peerId = null;

    this.socket.onopen = () => {
      console.log(`Socket connected to ${this.url}`);
    };

    this.socket.onmessage = async (e) => {
      console.log("Socket receive message: ", e.data);
      const prasedData = JSON.parse(e.data);

      if (prasedData.type === "INITIAL_CONNECTION") {
        document.querySelector(".userid").textContent = prasedData.data;
      } else if (prasedData.type === "PEER_CONNECTION_REQUEST") {
        this.peerId = prasedData.userId;

        const offer = prasedData.data;
        await rtc.setRemoteDescription(offer);

        const answer = await rtc.createAnswer();
        await rtc.setLocalDescription(answer);

        this.send({
          type: "PEER_CONNECTION_RESPONSE",
          userId: this.peerId,
          data: answer,
        });
      } else if (prasedData.type === "PEER_CONNECTION_RESPONSE") {
        this.peerId = prasedData.userId;

        const answer = prasedData.data;
        if (!rtc.currentRemoteDescription) {
          await rtc.setRemoteDescription(answer);
        }
      } else if (prasedData.type === "ICE_CANDIDATE") {
        try {
          await rtc.addIceCandidate(prasedData.data);
        } catch (err) {
          console.error("Error adding ICE candidate:", err);
        }
      }
    };

    this.socket.onclose = () => console.log("Socket closed");
    this.socket.onerror = () => {
      console.log("Socket error occurs, closing connection");
      this.socket.close();
    };

    WebSocketSingleton.instance = this;
  }

  send(data) {
    if (this.socket.readyState === WebSocket.OPEN) {
      this.socket.send(JSON.stringify(data));
    } else {
      throw new Error("WebSocket is not open");
    }
  }
}

let localStream, remoteStream;

window.addEventListener("load", async () => {
  try {
    const ws = new WebSocketSingleton(SOCKET_CONNECTION_URL);

    rtc.onicecandidate = (event) => {
      if (event.candidate && ws.peerId) {
        ws.send({
          type: "ICE_CANDIDATE",
          userId: ws.peerId,
          data: event.candidate,
        });
      }
    };

    rtc.oniceconnectionstatechange = () => {
      console.log("ICE state:", rtc.iceConnectionState);
    };

    document
      .querySelector(".action__button")
      .addEventListener("click", async () => {
        const targetId = document.querySelector(".action__input").value.trim();
        if (!targetId) return alert("Enter a target user id");

        ws.peerId = targetId;

        const offer = await rtc.createOffer();
        await rtc.setLocalDescription(offer);

        ws.send({
          type: "PEER_CONNECTION_REQUEST",
          data: offer,
          userId: targetId,
        });
      });

    localStream = await navigator.mediaDevices.getUserMedia({
      video: true,
      audio: false,
    });
    remoteStream = new MediaStream();

    document.querySelector(".stream__local").srcObject = localStream;
    document.querySelector(".stream__remote").srcObject = remoteStream;

    localStream
      .getTracks()
      .forEach((track) => rtc.addTrack(track, localStream));

    rtc.ontrack = (event) => {
      event.streams[0].getTracks().forEach((track) => {
        remoteStream.addTrack(track);
      });
    };
  } catch (err) {
    console.error(err);
    alert(err);
  }
});

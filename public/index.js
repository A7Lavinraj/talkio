const SOCKET_CONNECTION_URL = "ws://localhost:8080/ws";

const rtc = new RTCPeerConnection({
  iceCandidatePoolSize: 10,
  iceServers: [
    {
      urls: ["stun:stun1.l.google.com:19302", "stun:stun2.l.google.com:19302"],
    },
  ],
});

class WebSocketSingleton {
  constructor(url) {
    if (WebSocketSingleton.instance) {
      return WebSocketSingleton.instance;
    }

    this.url = url;
    this.socket = new WebSocket(url);
    if (!this.socket) {
      throw new Error("Unable to create Socket");
    }

    this.socket.onopen = () => {
      console.log(`Socket connected to ${this.url}`);
    };

    this.socket.onmessage = async (e) => {
      console.log("Socket receive message: ", e.data);

      const prasedData = JSON.parse(e.data);
      if (prasedData.type === "INITIAL_CONNECTION") {
        document.querySelector(".userid").innerHTML = prasedData.data;
      } else if (prasedData.type === "PEER_CONNECTION_REQUEST") {
        console.log(prasedData.data);
        const isPermissionGrated = confirm(
          "A request has arrived, You wanna connect?",
        );

        if (isPermissionGrated) {
          let offer = prasedData.data;
          await rtc.setRemoteDescription(offer);

          let answer = await rtc.createAnswer();
          await rtc.setLocalDescription(answer);

          this.send({
            type: "PEER_CONNECTION_RESPONSE",
            userId: prasedData.userId,
            data: answer,
          });
        }
      } else if (prasedData.type === "PEER_CONNECTION_RESPONSE") {
        console.log("Add answer triggerd");

        let answer = prasedData.data;
        console.log("answer:", answer);

        if (!rtc.currentRemoteDescription) {
          rtc.setRemoteDescription(answer);
        }
      }
    };

    this.socket.onclose = () => {
      console.log("Socket closed");
    };

    this.socket.onerror = () => {
      console.log("Socket error occurs, closing connection");
      this.socket.close();
    };

    WebSocketSingleton.instance = this;
    Object.freeze(this);
  }

  send(data) {
    if (this.socket.readyState === WebSocket.OPEN) {
      this.socket.send(JSON.stringify(data));
    } else {
      throw new Error("WebSocket is not open");
    }
  }
}

window.addEventListener("load", () => {
  try {
    const ws = new WebSocketSingleton(SOCKET_CONNECTION_URL);
    document
      .querySelector(".action__button")
      .addEventListener("click", async () => {
        const offer = await rtc.createOffer();
        await rtc.setLocalDescription(offer);
        ws.send({
          type: "PEER_CONNECTION_REQUEST",
          data: offer,
          userId: document.querySelector(".action__input").value,
        });
      });
  } catch (err) {
    console.error(err);
  }
});

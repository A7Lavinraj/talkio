const SOCKET_CONNECTION_URL = "ws://localhost:8080/ws";

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

    this.socket.onmessage = (e) => {
      console.log("Socket receive message: ", e.data);

      const prasedData = JSON.parse(e.data);
      if (prasedData.type === "INITIAL_CONNECTION") {
        document.querySelector(".userid").innerHTML = prasedData.data;
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
      this.socket.send(data);
    } else {
      throw new Error("WebSocket is not open");
    }
  }
}

window.addEventListener("load", () => {
  try {
    new WebSocketSingleton(SOCKET_CONNECTION_URL);
  } catch (err) {
    console.error(err);
  }
});

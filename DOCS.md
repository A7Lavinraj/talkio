# WebRTC - Web Real Time Communication

**WebRTC** is a free and open-source project providing web browsers and mobile applications with real-time communication via application programming interfaces.

## What is needed to establish connection?

To establish connection between two computers, all we need there IP addresses.

## Naive approach to establish connection?

1. First we open a **WebSocket** connection.
2. Then connect both computers to that **WebSocket** connection.
3. Then brodcast information(IP address and more) from one of the computers.
4. In response other computer will response with it's information.
5. Hence, Both computers have each other information.
6. Close **WebSocket** connection, Although it's completely fine to leave it open for further connections.
7. Now we can establish a **WebRTC** connection between both the computers.

## What is wrong with **Naive** approach?

1. As you can clearly see that we are creating a new **WebRTC** connection whenever a person joins the room.
2. Here if try to transfer data(audio or video) through this connection, then following happens:
   - Let say we have `N` person in the room.
   - Then our client have to send data to all other `N - 1` person. Which cause heavy traffic problems.
   - At this traffic only **5 to 10** connection are enough to crash a good specs computer.

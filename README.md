Chatroom client + server.  

# client

    client <name> [serverIP]  

The default server IP is `localhost:5000`.  
Clients should choose a unique name or they cannot join.  

# server

    server [port]

The default port is 5000.  
The commands supported are:  
- `/kick <name>`: kick client with name
- `list`: list all online clients
- `/kickall`: kick everyone

![](https://i.imgur.com/egLX2HT.png)
![](https://i.imgur.com/UcjHJJD.png)
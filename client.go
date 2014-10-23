package main

import ()

/*
Enter Address: sean@cotap.com
Enter Name: Sean Pont
Commands:
  list: list conversations
  create: <title> create and join a conversation with specified title
  say <message>: Say something in the current conversation
  leave: leave the current conversation
  exit: exit the program (and leave the current conversation)
$ list
<lists conversations>
$ join <conversation name>
<shows last 10 messages>
<messages appear as they arrive>
$ say Hi guys
Sean: Hi guys
$ leave
$
*/

type TcpTapClient struct {
	service string
	address string
}

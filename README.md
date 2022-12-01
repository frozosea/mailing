# mailing

This is super simple package for sending mails with 2 services.

- [https://www.unisender.com/](Unisender)
- [https://elasticemail.com/](ElasticEmail)

## Usage

````go
package main

import (
	"github.com/frozosea/mailing"
	"golang.org/x/net/context"
	"log"
)

func main() {

	elastic, err := mailing.NewWithElasticEmail("your smtp host", 123, "send email", "password for email", "api-auth-key for elasticEmail", "list name for contacts")
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	//Send email with elastic without attachment
	err = elastic.SendSimple(ctx, []string{"example@email.org"}, "your subject", "text in email", "content type (text/html...)")
	if err != nil {
		log.Fatal(err)
	}

	//Send email with elastic with attachment
	err = elastic.SendWithFile(ctx, []string{"example@email.org"}, "your subject", "filepath to attachment")
	if err != nil {
		log.Fatal(err)
	}

	unisender := mailing.NewWithUniSender("sender name", "sender@mail.com", "unisedner-auth-api-key", "example signature in email")

	//Send email with elastic without attachment
	err = unisender.SendSimple(ctx, []string{"example@email.org"}, "your subject", "text in email", "content type (text/html...)")
	if err != nil {
		log.Fatal(err)
	}

	//Send email with elastic with attachment
	err = unisender.SendWithFile(ctx, []string{"example@email.org"}, "your subject", "filepath to attachment")
	if err != nil {
		log.Fatal(err)
	}
}


````

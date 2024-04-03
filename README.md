# Go-Ecommerce Subscription

This is an ecommerce project built using Go, integrating Stripe APIs to set up credit card transactions and handling monthly subscription packages.
The Stripe implementation focuses primarily on calling the Stripe API from the backend to create a Payment Intent to authorize and make transactions.

## Functionality

```
- Allow users to purchase single product
- Allow users to purchase a recurring monthly Stripe Plan
- Handling cancellations and refunds
- Save transaction information to Postgres DB
- Secure session authentication to front end
- Secure access to backend APIs through stateful tokens
- User management and access, instant logout using websockets
- Allow users to reset passwords safely and securely
- Micorservice that accepts JSON payload of individual purchase, produces PDF invoice,
create/attach PDF invoice and send email
```

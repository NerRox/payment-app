curl -X POST localhost:8080/enroll -d '{"userId": 10, "userBalance": 100}'
curl -X POST localhost:8080/withdraw -d '{"userId": 10, "userBalance": 1000}'
curl -X POST localhost:8080/transfer -d '{"SenderUserID": 10, "receiverUserId": 15, "amount": 10}'

curl localhost:8080/balance?id=10

curl localhost:8080/balance?id=10&currency=USD
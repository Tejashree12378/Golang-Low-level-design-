# Rate Limiter
### Problem Statement: 
Implement a rate limiter which limits number of requests each user can send
to the server in a minute. 

### Explanation of the approach:
1. We need to implement a cache with ttl
2. I used map with userID as key, counter (count and expireAt) as value
3. for every request fetch counter and validate and handle accordingly
4. but we need to handle ttl as well, so for this i appended every processed
   request to a slice and used a go routine to clean this expired records every minute 
   to free up the space and remove invalid records (cache invalidation)
5. Now since we have 2 go routines(main and cleanup), where both are accessing the same variables
   we will have race condition and end up have data inconsistencies, so i used locks to 
   synchronise between them.
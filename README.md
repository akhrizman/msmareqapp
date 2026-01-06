# MSMA Testing Requirements Handbook

### Description
> This MVP web app was created for MSMA to enable students of the school to 
> view their testing and form requirements for attaining their next rank, as 
> well as previous ranks.  Admin users update student ranks, grant or revoke 
> access to the site, and control which requirements users can view (only the 
> requirements for the next rank by default)

### Basic deployment steps
```
docker build --no-cache -t msmareqapp .
docker run -dp 9088:8080 --name msmareqapp_dev msmareqapp
```
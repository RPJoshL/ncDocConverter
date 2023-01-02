# ncDocConverter

A Go program able to convert documents automatically to PDF / EPUB Files.

Currently, the following sources for documents are supported:

* Nextcloud with OnlyOffice
* Boockstack

As a destination to save the converted files only **Nextcloud** is supported.


## Setting it up

For using the 


### BookStack

For converting books of BookStack you need to create an API token for the user to access the books:
1. Login as Admin
2. Go to *Settings → Users*
3. Select user for API access
4. Scroll down to `API Tokens` and click `CREATE TOKEN`
5. Set a name and expire date. Click `save`
6. Copy the ID and Token. The field `apiToken` will contain the combination from `id:token`

Now you need also create a new role or edit an existing role.
1. Go to *Settings → Roles*
2. Edit and existing Role (the role which the user have) or create a new role
3. Check the box `Access system API` and `Export content` in `System permissions`
4. Assing View Role *(all and own)* for *Shelves, Books, Chapters and Pages* 

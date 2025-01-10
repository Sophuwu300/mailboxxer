# mailboxxer

An application to view your postfix inbox.

## Features

- Parses headers, body, and attachments from emails
- Saves metadata in a SQLite database for easy searching
- Saves attachments and bodies in a directory for easy access
- CLI for viewing emails
  - Parses HTML and text emails to display in terminal
  - Doesn't show attachments or inline images
- Web interface for viewing emails
    - Parses HTML and text emails to display in browser
    - Removes script tags from HTML emails to prevent XSS
    - Display HTML inside an iframe to prevent XSS
    - Displays inline images
    - Shows attachments

## Prerequisites

Set up postfix to save emails to `$HOME/.mailbox/inbox/` and run the following command to create the database:


## Flags

`--cli` runs the cli interface

`--web` runs the webinterface on port 4131

If none are given, will parse and save new emails, then exit.


## License
MIT

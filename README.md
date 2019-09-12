# flex-sftp

Flexible and pluggable SFTP server that you can plug stuff into

## Warning

We included a private/public key-pair in the `keys`
directory. These should *NOT* be used in production
because they have been exposed "in plain text" in
this repository and can be copied by anyone to
masquerade as your server should you use these keys
as private keys for your server. Instead, generate
a strong key and expire them to ensure the safety
of your clients' sessions.

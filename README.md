# Musify Telegram Bot
A simple inline music bot for [Telegram](t.me) based on vk.com music API

## Deploy
This project's main branch is `heroku` as it's intended to deploy as a heroku app. This is the most convenient and fastest way to get you started. You are free to launch it manually if you wish so, though.

[![Deploy](https://www.herokucdn.com/deploy/button.svg)](https://heroku.com/deploy)

## Environmental Variables
- TELEGRAM_BOT_API_TOKEN  
**required**  
Telegram Bot token received from the Telegram's [BotFather](https://t.me/botfather). To get started with Telegram Bots: https://core.telegram.org/bots

- VK_USERNAME  
**required**  
Your vk.com account's username (mobile phone or email). This remains totally secret inside your heroku deployment. VK Music API is accessed using some real user credentials.

- VK_PASSWORD  
**required**  
Your vk.com account's password. This remains totally secret inside your heroku deployment. VK Music API is accessed using some real user credentials.

- VK_API_ACCESS_TOKEN  
[Service token](https://vk.com/dev/access_token) of any app on vk.com. Used for recognition of vk.com nicknames e.g. @durov (Extra bot feature: one can use their VK's playlist in Telegram using the bot if their music is public). You can create a dummy **standalone** VK's application [here](https://vk.com/editapp?act=create) and use its service (not to confuse with account token) token. This functionality will not be available if you leave this variable empty

- TELEGRAM_OWNER_CHAT_ID  
**required**  
ChatID of your account in Telegram. Bot will send you some error logs as well as captcha inquires in an unlikely case it is needed to login on vk.com. [How to get your Chat ID](https://sean-bradley.medium.com/get-telegram-chat-id-80b575520659)

- HAPPIDEV_API_TOKEN  
[Happi.dev](https://happi.dev/docs/music) API token. Used for lyrics search. Simply register on happi.dev (it's free) to get a token. This functionality will not be available if you leave this variable empty

- AUDD_API_TOKEN
[Audd Music Recognition]() API token. Used for music recognition by voice messages and for lyrics search. Simply register on audd.io (free indie plan) to get a token. This functionality will not be available if you leave this variable empty

- TELEGRAM_RHASH  
Telegram [InstantView](https://instantview.telegram.org/) template rhash. Used for sending lyrics in InstantView form. Only useful if Happi.dev API token is provided. Here's [how to](https://instantview.telegram.org/#templates-tutorial) deal with instantviews. [Example](https://t.me/iv?url=https%3A%2F%2Fmusify-bot.herokuapp.com%2Flyrics%2F1996%2F2382%2F40356&rhash=81c30e9e431429). This functionality will not be available if you leave this variable empty

- MUSIFY_SQL_DSN  
MySQL DSN value of your own MySQL server. Used for storing some metadata and astatistics. See below in ethe readme. Also used to buffer lyrics. This functionality will not be available if you leave this variable empty

## Miscellenious
### A note on vk.com account use
MusifyBot makes use vk.com music API through its `al_audio.php` interface. A user account is needed to make calls to it, that is why VK_USERNAME and VK_PASSWORD are required in the environment.  
> THESE CREDENTIALS ARE NOT STORED ANYWHERE OUTSIDE YOUR DEPLOYMENT/MACHINE/ENVIRONMENT.

Note that the behavior is different based on the bot's server location:

- bot server is located outside Russia (this is the case for heroku servers):  
Your account must have a valid subscription on VK's [Boom](https://vk.com/boom) music service in order to work

- bot server is located in Russia (you will need to host it manually using, for example, [Yandex.Cloud](https://cloud.yandex.com/) or other VM provider or host the bot on your own server in Russia):   
No Boom subscription is required

### Statistics DB
To gather statistics on bot's activity, an instance of MySQL or MariaDB must be up and running somewhere on your own servers.

Launch such a database and execute the `musify-db.sql` script to initialize the schema compatible to this bot. Probably, the easiest way would be to launch a [dockerized MariaDB](https://hub.docker.com/_/mariadb). Please read the instructions from the link on how to properly set environmental variables.

After you've set the DB up, you must now be able to construct your database conncetion DSN that you should put into the `MUSIFY_SQL_DSN` config var of your heroku deployment, e.g.:

```
<MYSQL_USER>:<MYSQL_PASSWORD>@tcp(<public-ip-address>:3306)/<MYSQL_DATABASE>
```

Note that variables names in the DSN above are the same as their corresponding environmental variables of this [docker container](https://hub.docker.com/_/mariadb)
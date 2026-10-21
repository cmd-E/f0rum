CREATE TABLE users_new (
 id INTEGER PRIMARY KEY, username TEXT NOT NULL UNIQUE, email TEXT NOT NULL UNIQUE,
 password TEXT NOT NULL, role TEXT NOT NULL DEFAULT 'user' CHECK(role IN ('user','admin'))
);
CREATE TABLE sessions_new (
 token_hash TEXT PRIMARY KEY, user_id INTEGER NOT NULL REFERENCES users_new(id) ON DELETE CASCADE,
 created_at INTEGER NOT NULL, expires_at INTEGER NOT NULL
);
CREATE TABLE threads_new (id INTEGER PRIMARY KEY, title TEXT NOT NULL, description TEXT NOT NULL);
CREATE TABLE posts_new (
 id INTEGER PRIMARY KEY, title TEXT NOT NULL, content TEXT NOT NULL,
 likes INTEGER NOT NULL DEFAULT 0, dislikes INTEGER NOT NULL DEFAULT 0,
 authorid INTEGER NOT NULL REFERENCES users_new(id)
);
CREATE TABLE postsandcats_new (
 id INTEGER PRIMARY KEY, threadid INTEGER NOT NULL REFERENCES threads_new(id) ON DELETE CASCADE,
 postid INTEGER NOT NULL REFERENCES posts_new(id) ON DELETE CASCADE, UNIQUE(threadid,postid)
);
CREATE TABLE comments_new (
 id INTEGER PRIMARY KEY, postid INTEGER NOT NULL REFERENCES posts_new(id) ON DELETE CASCADE,
 user_id INTEGER NOT NULL REFERENCES users_new(id),
 content TEXT NOT NULL, likes INTEGER NOT NULL DEFAULT 0, dislikes INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE rtactions_new (
 id INTEGER PRIMARY KEY, postid INTEGER NOT NULL REFERENCES posts_new(id) ON DELETE CASCADE,
 user_id INTEGER NOT NULL REFERENCES users_new(id), type TEXT NOT NULL CHECK(type IN ('like','dislike')),
 UNIQUE(user_id,postid)
);
CREATE TABLE rtactionscomm_new (
 id INTEGER PRIMARY KEY, commid INTEGER NOT NULL REFERENCES comments_new(id) ON DELETE CASCADE,
 user_id INTEGER NOT NULL REFERENCES users_new(id), type TEXT CHECK(type IN ('like','dislike')),
 UNIQUE(user_id,commid)
);

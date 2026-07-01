const express = require('express');
const session = require('express-session');
const Database = require('better-sqlite3');

const app = express();
const db = new Database('database.db');

const PORT = process.env.PORT || 7860;
const OWNER_ID = "8135766988";

app.use(express.json());
app.use(session({
  secret: 'secret',
  resave: false,
  saveUninitialized: true
}));

// DB init
db.exec(`
CREATE TABLE IF NOT EXISTS logins (
  telegram_id TEXT,
  token TEXT,
  exp INTEGER
);
`);

db.exec(`
CREATE TABLE IF NOT EXISTS users (
  telegram_id TEXT,
  role TEXT DEFAULT 'user'
);
`);

// AUTH
app.get('/auth', (req, res) => {
  const token = req.query.token;

  const row = db.prepare(`
    SELECT * FROM logins
    WHERE token=? AND exp > ?
  `).get(token, Date.now());

  if (!row) return res.send("❌ Invalid or expired link");

  req.session.user = row.telegram_id;

  db.prepare("INSERT OR IGNORE INTO users (telegram_id, role) VALUES (?, 'user')")
    .run(row.telegram_id);

  db.prepare("DELETE FROM logins WHERE token=?").run(token);

  res.send("✅ Logged in successfully");
});

// ADMIN CHECK
function isOwner(req) {
  return req.session.user === OWNER_ID;
}

// ADMIN PANEL
app.get('/admin', (req, res) => {
  if (!isOwner(req)) return res.send("❌ No access");

  res.send(`
    <h1>👑 Admin Panel</h1>
    <p>Owner ID: ${OWNER_ID}</p>
  `);
});

app.listen(PORT, () => {
  console.log("Web running on " + PORT);
});

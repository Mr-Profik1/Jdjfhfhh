import os
import time
import uuid
import sqlite3
from telegram import Update
from telegram.ext import Application, CommandHandler, ContextTypes

TOKEN = os.getenv("8954009025:AAE3Tc0BYUYq7NkzjljW1s57v89VTkWS1c0")
SITE_URL = os.getenv("SITE_URL", "https://your-app.onrender.com")

OWNER_ID = "8135766988"

conn = sqlite3.connect("database.db", check_same_thread=False)
cur = conn.cursor()

cur.execute("""
CREATE TABLE IF NOT EXISTS logins (
    telegram_id TEXT,
    token TEXT,
    exp INTEGER
)
""")

cur.execute("""
CREATE TABLE IF NOT EXISTS users (
    telegram_id TEXT PRIMARY KEY,
    role TEXT DEFAULT 'user'
)
""")

conn.commit()

def is_owner(user_id):
    return str(user_id) == OWNER_ID

async def start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    user_id = str(update.effective_user.id)

    # register user
    cur.execute("INSERT OR IGNORE INTO users (telegram_id, role) VALUES (?, 'user')", (user_id,))
    conn.commit()

    token = str(uuid.uuid4())
    exp = int(time.time()) + 600

    cur.execute(
        "INSERT INTO logins (telegram_id, token, exp) VALUES (?, ?, ?)",
        (user_id, token, exp)
    )
    conn.commit()

    link = f"{SITE_URL}/auth?token={token}"

    await update.message.reply_text(
        f"🔐 Одноразовая ссылка:\n{link}\n⏳ 10 минут"
    )

async def admin(update: Update, context: ContextTypes.DEFAULT_TYPE):
    if not is_owner(update.effective_user.id):
        return await update.message.reply_text("❌ No access")

    await update.message.reply_text("👑 Admin panel active")

def main():
    app = Application.builder().token(TOKEN).build()

    app.add_handler(CommandHandler("start", start))
    app.add_handler(CommandHandler("admin", admin))

    print("Bot running")
    app.run_polling()

if name == "__main__":
    main()

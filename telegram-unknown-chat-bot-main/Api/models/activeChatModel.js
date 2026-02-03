const mongoose = require("mongoose");

// Main Mongoose Schema for Active Chat
const activeChatSchema = new mongoose.Schema({
  telegramId_1: {
    type: String,
    require: true,
    unique: true,
  },
  telegramId_2: {
    type: String,
    require: true,
    unique: true,
  },
});

const ActiveChat = mongoose.model("ActiveChat", activeChatSchema);

module.exports = ActiveChat;

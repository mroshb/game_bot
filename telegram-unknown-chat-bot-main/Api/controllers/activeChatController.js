const ActiveChat = require("../models/activeChatModel");
const { createError } = require("../middlewares/errors");

// POST - /admin/activechats/create - creates a new active chat
exports.createActiveChat = async (req, res, next) => {
  try {
    let { telegramId_1, telegramId_2 } = req.body;

    if (!telegramId_1 || !telegramId_2) {
      throw createError(400, "", "bad request");
    }

    await ActiveChat.create({ telegramId_1, telegramId_2 });

    res.status(200).json({ message: "done" });
  } catch (err) {
    next(err);
  }
};

// GET - /admin/activechats/find/:id - finds a single active chat
exports.findActiveChat = async (req, res, next) => {
  try {
    const activeChat_1 = await ActiveChat.findOne({
      telegramId_1: req.params.id,
    });

    const activeChat_2 = await ActiveChat.findOne({
      telegramId_2: req.params.id,
    });

    if (!activeChat_1 && !activeChat_2) {
      throw createError(404, "", "no active chat found");
    } else if (activeChat_1) {
      res.status(200).json({ activeChat: activeChat_1 });
    } else if (activeChat_2) {
      res.status(200).json({ activeChat: activeChat_2 });
    }
  } catch (err) {
    next(err);
  }
};

// DELETE - /admin/activechats/delete/:id - deletes a active chat
exports.deleteActiveChat = async (req, res, next) => {
  try {
    const activeChat_1 = await ActiveChat.findOne({
      telegramId_1: req.params.id,
    });

    const activeChat_2 = await ActiveChat.findOne({
      telegramId_2: req.params.id,
    });

    if (!activeChat_1 && !activeChat_2) {
      throw createError(404, "", "no active chat found");
    } else if (activeChat_1) {
      await ActiveChat.findOneAndRemove({ telegramId_1: req.params.id });
    } else if (activeChat_2) {
      await ActiveChat.findOneAndRemove({ telegramId_2: req.params.id });
    }
    res.status(200).json({ message: "done" });
  } catch (err) {
    next(err);
  }
};

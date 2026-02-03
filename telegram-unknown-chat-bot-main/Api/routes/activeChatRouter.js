const express = require("express");

const activeChatController = require("../controllers/activeChatController");

const router = express.Router();

// POST - /admin/activechats/create - creates a new active chat
router.post("/create", activeChatController.createActiveChat);

// GET - /admin/activechats/find/:id - finds a single active chat
router.get("/find/:id", activeChatController.findActiveChat);

// DELETE - /admin/activechats/delete/:id - deletes a active chat
router.delete("/delete/:id", activeChatController.deleteActiveChat);

module.exports = router;

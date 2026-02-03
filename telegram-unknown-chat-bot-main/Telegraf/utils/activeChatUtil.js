const { default: axios } = require("axios");

exports.findActiveChat = async (id) => {
  try {
    const activeChat = await axios.get(
      `http://localhost:3000/admin/activechats/find/${id}`
    );

    return activeChat.data.activeChat;
  } catch (err) {
    return false;
  }
};

exports.createActiveChat = async (telegramId_1, telegramId_2) => {
  try {
    const result = await axios.post(
      "http://localhost:3000/admin/activechats/create",
      { telegramId_1, telegramId_2 }
    );

    return result;
  } catch (err) {
    return { data: { activeChat: null, message: null } };
  }
};

exports.deleteActiveChat = async (id) => {
  try {
    const activeChat = await axios.delete(
      `http://localhost:3000/admin/activechats/delete/${id}`
    );

    return true;
  } catch (err) {
    return false;
  }
};

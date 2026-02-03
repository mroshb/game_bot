const { findActiveChat } = require("./activeChatUtil");
const { findLookingUser } = require("./lookingUserUtil");

exports.isUserInWizard = async (ctx, next) => {
  const activechat = await findActiveChat(ctx.message.from.id);
  const lookingUser = await findLookingUser(ctx.message.from.id);
  const isInWizard = Boolean(ctx.session.wizard);
  if (lookingUser || activechat || ctx.scene.current || isInWizard) {
    return ctx.reply('شما در حال حاضر مشغول به یک عملیات دیگر هستید');
  } else {
    return next();
  }
};

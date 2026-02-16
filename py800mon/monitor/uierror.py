import time

from ..actions import Actions
from ..app import Component
from .appstate import state


class UIErrorTimeout(Component):
    async def update(self):
        if not state.ui_error:
            return False
        deadline = state.ui_error_deadline
        if deadline and time.monotonic() >= deadline:
            self.app.dispatch_action(Actions.CLEAR_UI_ERROR)
            return True
        return False

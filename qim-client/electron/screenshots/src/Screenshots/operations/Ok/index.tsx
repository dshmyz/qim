import type { ReactElement } from 'react';
import { useCallback } from 'react';
import composeImage from '../../composeImage';
import useCall from '../../hooks/useCall';
import useHistory from '../../hooks/useHistory';
import useReset from '../../hooks/useReset';
import useStore from '../../hooks/useStore';
import ScreenshotsButton from '../../ScreenshotsButton';

export default function Ok(): ReactElement {
  const { url, image, width, height, history, bounds, lang } = useStore();
  const [, historyDispatcher] = useHistory();
  const call = useCall();
  const reset = useReset();

  const onClick = useCallback(() => {
    historyDispatcher.clearSelect();
    setTimeout(() => {
      if (!bounds) {
        return;
      }
      composeImage({
        image,
        url,
        width,
        height,
        history,
        bounds,
      }).then((blob) => {
        call('onOk', blob, bounds);
        reset();
      });
    });
  }, [
    historyDispatcher,
    image,
    url,
    width,
    height,
    history,
    bounds,
    call,
    reset,
  ]);

  return (
    <ScreenshotsButton
      title={lang.operation_ok_title}
      icon="icon-ok"
      onClick={onClick}
    />
  );
}

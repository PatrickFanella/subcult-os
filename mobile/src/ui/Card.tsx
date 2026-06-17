import type { PropsWithChildren } from 'react';
import { Pressable, View, type PressableProps } from 'react-native';

type CardProps = PropsWithChildren<PressableProps & {
  pressable?: boolean;
  contentClassName?: string;
}>;

export function Card({ children, className, contentClassName, pressable = false, ...pressableProps }: CardProps) {
  const content = <View className={['panel-inner', contentClassName].filter(Boolean).join(' ')}>{children}</View>;

  if (pressable || pressableProps.onPress) {
    return (
      <Pressable {...pressableProps} className={['panel active:opacity-80', className].filter(Boolean).join(' ')}>
        {content}
      </Pressable>
    );
  }

  return <View className={['panel', className].filter(Boolean).join(' ')}>{content}</View>;
}

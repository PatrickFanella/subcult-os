import type { PropsWithChildren } from 'react';
import { SafeAreaView } from 'react-native-safe-area-context';

export function Screen({ children }: PropsWithChildren) {
  return <SafeAreaView className="screen">{children}</SafeAreaView>;
}

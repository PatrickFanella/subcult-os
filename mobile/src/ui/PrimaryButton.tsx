import { Pressable, Text } from 'react-native';

type PrimaryButtonProps = {
  label: string;
  onPress?: () => void;
};

export function PrimaryButton({ label, onPress }: PrimaryButtonProps) {
  return (
    <Pressable onPress={onPress} className="btn-primary active:opacity-80">
      <Text className="text-fg-inverse text-body font-black">{label}</Text>
    </Pressable>
  );
}

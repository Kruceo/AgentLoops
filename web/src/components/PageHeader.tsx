interface PageHeaderProps {
  title: string;
  description?: string;
  children?: JSX.Element;
}

export default function PageHeader(props: PageHeaderProps) {
  return (
    <div class="flex items-center justify-between mb-6">
      <div>
        <h2 class="text-2xl font-bold text-white">{props.title}</h2>
        {props.description && (
          <p class="text-gray-400 text-sm mt-1">{props.description}</p>
        )}
      </div>
      <div class="flex items-center gap-2 text-sm text-gray-400">
        {props.children}
      </div>
    </div>
  );
}

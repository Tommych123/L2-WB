package main

// or объединяет несколько done-каналов в один, возвращаемый канал закрывается, как только закроется любой из входных каналов
func or(channels ...<-chan interface{}) <-chan interface{} {

	if len(channels) == 0 {
		return nil
	}

	if len(channels) == 1 {
		return channels[0]
	}

	// Итоговый канал
	orDone := make(chan interface{})

	// Запускаем горутину которая будет ждать закрытия любого канала
	go func() {
		// Как только select сработает закрываем результирующий канал
		defer close(orDone)

		switch len(channels) {

		case 2:
			select {
			case <-channels[0]:
			case <-channels[1]:
			}

		// Если каналов больше двух используем рекурсивное разбиение пополам
		default:
			// Делим список каналов на две группы
			mid := len(channels) / 2

			// Рекурсивно создаём два or-канала для каждой половины и ждём какой из них сработает первым
			select {
			case <-or(channels[:mid]...):
			case <-or(channels[mid:]...):
			}
		}
	}()

	return orDone
}
